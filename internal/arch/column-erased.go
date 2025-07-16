package arch

import (
	"math"
	"reflect"
	"unsafe"
)

type ErasedColumn struct {
	ComponentType *ComponentType
	Type          reflect.Type

	itemSize uintptr

	// keeps track of added + changed values
	added   []Tick
	changed []Tick

	// keeps track of the latest hash value of a component
	// for comparable components
	hashes []HashValue

	// slice of values
	slice reflect.Value

	// capacity and length of the slice
	len, cap int

	// memory points to the data of an unsafe slice of component instances
	memory unsafe.Pointer

	dummyValue ErasedComponent

	trivialCopy bool
}

func MakeErasedColumn(ty *ComponentType) MakeColumn {
	return func() *ErasedColumn {
		sliceType := reflect.SliceOf(ty.Type)
		slice := reflect.New(sliceType).Elem()

		return &ErasedColumn{
			ComponentType: ty,
			Type:          ty.Type,
			itemSize:      ty.Type.Size(),
			slice:         slice,
			len:           slice.Len(),
			cap:           slice.Cap(),
			memory:        slice.UnsafePointer(),
			trivialCopy:   !ty.HasPointers,
			dummyValue:    ty.New(),
		}
	}
}

type buf *[math.MaxInt32]byte

func (e *ErasedColumn) ptrTo(row Row) unsafe.Pointer {
	if int(row) >= e.len {
		panic("out of bounds")
	}

	return unsafe.Add(e.memory, uintptr(row)*e.itemSize)
}

func (e *ErasedColumn) copyValueTo(row Row, value ErasedComponent) {
	target := buf(e.ptrTo(row))
	source := buf(pointerTo(value))
	copy((*target)[:e.itemSize], (*source)[:e.itemSize])
}

func (e *ErasedColumn) Append(tick Tick, component ErasedComponent) {
	//assert.IsPointerType(reflect.TypeOf(component))

	e.ensureSpace()

	rowTarget := Row(e.len)

	e.len += 1

	if e.trivialCopy {
		e.copyValueTo(rowTarget, component)
	} else {
		target := e.ptrTo(rowTarget)
		e.ComponentType.UnsafeSetValue(target, component)
	}

	e.added = append(e.added, tick)
	e.changed = append(e.changed, tick)
	e.hashes = append(e.hashes, e.hash(rowTarget))
}

func (e *ErasedColumn) Copy(from, to Row) {
	e.added[to] = e.added[from]
	e.changed[to] = e.changed[from]
	e.hashes[to] = e.hashes[from]

	source := e.ptrTo(from)
	target := e.ptrTo(to)

	if e.trivialCopy {
		copy((*buf(target))[:e.itemSize], (*buf(source))[:e.itemSize])
	} else {
		e.ComponentType.UnsafeCopyValue(target, source)
	}
}

func (e *ErasedColumn) Truncate(n Row) {
	e.added = e.added[:n]
	e.changed = e.changed[:n]
	e.hashes = e.hashes[:n]

	e.len = int(n)
}

func (e *ErasedColumn) Get(row Row) ErasedComponent {
	return packValue(e.dummyValue, e.ptrTo(row))
}

func (e *ErasedColumn) Added(row Row) Tick {
	return e.added[row]
}

func (e *ErasedColumn) Changed(row Row) Tick {
	return e.changed[row]
}

func (e *ErasedColumn) Update(tick Tick, row Row, cv ErasedComponent) {
	e.changed[row] = tick

	if e.trivialCopy {
		e.copyValueTo(row, cv)
	} else {
		e.ComponentType.UnsafeSetValue(e.ptrTo(row), cv)
	}

	e.hashes[row] = e.hash(row)
}

func (e *ErasedColumn) Import(sourceColumn *ErasedColumn, row Row) {
	e.added = append(e.added, sourceColumn.added[row])
	e.changed = append(e.changed, sourceColumn.changed[row])
	e.hashes = append(e.hashes, sourceColumn.hashes[row])

	e.ensureSpace()

	rowTarget := Row(e.len)

	e.len += 1

	source := sourceColumn.ptrTo(row)
	target := e.ptrTo(rowTarget)

	if e.trivialCopy {
		copy((*buf(target))[:e.itemSize], (*buf(source))[:e.itemSize])
	} else {
		e.ComponentType.UnsafeCopyValue(target, source)
	}
}

func (e *ErasedColumn) ensureSpace() {
	if e.cap == e.len {
		// need to allocate memory
		e.slice.SetLen(e.len)
		e.slice.Grow(max(16, e.len*2/3))
		e.memory = e.slice.UnsafePointer()
		e.cap = e.slice.Cap()
	}
}

func (e *ErasedColumn) CheckChanged(tick Tick) {
	if e.itemSize == 0 || e.len == 0 || !e.ComponentType.Comparable {
		return
	}

	n := Row(e.len)
	hashes, changed := e.hashes, e.changed

	_ = hashes[n-1]
	_ = changed[n-1]

	if e.ComponentType.TriviallyHashable {
		memorySlices := e.ComponentType.MemorySlices
		for row := range n {
			hashValue := hashOf(memorySlices, e.ptrTo(row))
			if hashes[row] != hashValue {
				changed[row] = tick
				hashes[row] = hashValue
			}
		}

		return
	}

	if maphash := e.ComponentType.Maphash; maphash != nil {
		for row := range n {
			hashValue := maphash(e.Get(row))
			if hashes[row] != hashValue {
				changed[row] = tick
				hashes[row] = hashValue
			}
		}

		return
	}
}

func (e *ErasedColumn) Len() int {
	return e.len
}

func (e *ErasedColumn) Access() ColumnAccess {
	return ColumnAccess{
		base:   e.memory,
		stride: e.itemSize,
	}
}

func (e *ErasedColumn) hash(row Row) HashValue {
	if !e.ComponentType.Comparable {
		return 0
	}

	if e.ComponentType.TriviallyHashable {
		return hashOf(e.ComponentType.MemorySlices, e.ptrTo(row))
	}

	if maphash := e.ComponentType.Maphash; maphash != nil {
		return maphash(e.Get(row))
	}

	return 0
}

func pointerTo(value ErasedComponent) unsafe.Pointer {
	// TODO guard with debug clause
	// assert.IsPointerType(reflect.ValueOf(value).Type())

	type iface struct{ typ, val unsafe.Pointer }
	return (*iface)(unsafe.Pointer(&value)).val
}

func packValue(tmpl ErasedComponent, newValue unsafe.Pointer) ErasedComponent {
	type iface struct{ typ, val unsafe.Pointer }

	// update pointer in interface value
	(*iface)(unsafe.Pointer(&tmpl)).val = newValue
	return tmpl
}
