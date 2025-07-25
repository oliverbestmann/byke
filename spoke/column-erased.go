package spoke

import (
	"bytes"
	"math"
	"reflect"
	"unsafe"
)

type ErasedColumn struct {
	ComponentType *ComponentType

	// callback that will be invoked whenever the columns data buffer
	// size was increased.
	OnGrow func()

	// capacity and length of the slice
	len, cap int

	// memory points to the data of an unsafe slice of component instances
	memory unsafe.Pointer

	itemSize uintptr

	// keeps track of added + changed values
	added   []Tick
	changed []Tick

	// used to speed up Added + Changed queries directly based
	// on archetype checks
	LastAdded   Tick
	LastChanged Tick

	// keeps track of the latest hash value of a component
	// for comparable components
	hashes []HashValue

	// slice of values
	slice reflect.Value

	// a second copy of the values used for dirty tracking
	sliceCopy []byte

	dummyValue ErasedComponent

	trivialCopy bool
}

func MakeErasedColumn(ty *ComponentType) MakeColumn {
	return func() *ErasedColumn {
		sliceType := reflect.SliceOf(ty.Type)
		slice := reflect.New(sliceType).Elem()

		return &ErasedColumn{
			ComponentType: ty,
			LastAdded:     NoTick,
			LastChanged:   NoTick,
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

func (e *ErasedColumn) shadowPtrTo(row Row) unsafe.Pointer {
	if int(row) >= e.len {
		panic("out of bounds")
	}

	base := unsafe.Pointer(unsafe.SliceData(e.sliceCopy))
	return unsafe.Add(base, uintptr(row)*e.itemSize)
}

func (e *ErasedColumn) copyValueTo(rowTarget Row, component ErasedComponent) {
	if e.trivialCopy {
		e.rawCopyValue(rowTarget, component)
	} else {
		ptrTarget := e.ptrTo(rowTarget)
		e.ComponentType.UnsafeSetValue(ptrTarget, component)
	}
}

func rawCopy(to, from unsafe.Pointer, size uintptr) {
	dst := (*buf(to))[:size]
	src := (*buf(from))[:size]
	copy(dst, src)
}

func (e *ErasedColumn) rawCopyValue(row Row, value ErasedComponent) {
	target := e.ptrTo(row)
	source := pointerTo(value)
	rawCopy(target, source, e.itemSize)
}

func (e *ErasedColumn) Append(tick Tick, component ErasedComponent) {
	//assert.IsPointerType(reflect.TypeOf(component))

	e.ensureSpace()

	rowTarget := Row(e.len)

	e.len += 1

	e.copyValueTo(rowTarget, component)

	e.added = append(e.added, tick)
	e.changed = append(e.changed, tick)
	e.hashes = append(e.hashes, e.hash(rowTarget))

	e.LastAdded = tick
	e.LastChanged = tick

	// copy the byte to the extra storage
	e.sliceCopy = append(e.sliceCopy, (*(buf)(e.ptrTo(rowTarget)))[:e.itemSize]...)
}

func (e *ErasedColumn) Copy(from, to Row) {
	e.added[to] = e.added[from]
	e.changed[to] = e.changed[from]
	e.hashes[to] = e.hashes[from]

	source := e.ptrTo(from)
	target := e.ptrTo(to)

	if e.trivialCopy {
		rawCopy(target, source, e.itemSize)
	} else {
		e.ComponentType.UnsafeCopyValue(target, source)
	}

	// copy the byte within the extra storage
	rawCopy(e.shadowPtrTo(to), e.shadowPtrTo(from), e.itemSize)
}

func (e *ErasedColumn) Truncate(n Row) {
	e.added = e.added[:n]
	e.changed = e.changed[:n]
	e.hashes = e.hashes[:n]

	e.len = int(n)

	e.sliceCopy = e.sliceCopy[:uintptr(n)*e.itemSize]
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
	e.LastChanged = tick

	e.copyValueTo(row, cv)
	e.hashes[row] = e.hash(row)

	// copy the new data into the shadow buffer
	rawCopy(e.shadowPtrTo(row), e.ptrTo(row), e.itemSize)
}

func (e *ErasedColumn) Import(sourceColumn *ErasedColumn, row Row) {
	e.added = append(e.added, sourceColumn.added[row])
	e.changed = append(e.changed, sourceColumn.changed[row])
	e.hashes = append(e.hashes, sourceColumn.hashes[row])

	e.LastAdded = max(e.LastAdded, sourceColumn.added[row])
	e.LastChanged = max(e.LastChanged, sourceColumn.changed[row])

	e.ensureSpace()

	rowTarget := Row(e.len)

	e.len += 1

	source := sourceColumn.ptrTo(row)
	target := e.ptrTo(rowTarget)

	if e.trivialCopy {
		rawCopy(target, source, e.itemSize)
	} else {
		e.ComponentType.UnsafeCopyValue(target, source)
	}

	// append the byte to the extra storage
	e.sliceCopy = append(e.sliceCopy, (*(buf)(e.ptrTo(rowTarget)))[:e.itemSize]...)
}

func (e *ErasedColumn) ensureSpace() {
	if e.cap == e.len {
		// need to allocate memory
		e.slice.SetLen(e.len)
		e.slice.Grow(max(16, e.len*2/3))
		e.memory = e.slice.UnsafePointer()
		e.cap = e.slice.Cap()

		if grow := e.OnGrow; grow != nil {
			grow()
		}
	}
}

func (e *ErasedColumn) CheckChanged(tick Tick) {
	if e.itemSize == 0 || e.len == 0 || !e.ComponentType.Comparable {
		return
	}

	if e.ComponentType.memcmp {
		e.checkChangesUsingSliceCompare(tick)
		return
	}

	n := Row(e.len)
	hashes, changed := e.hashes, e.changed

	_ = hashes[n-1]
	_ = changed[n-1]

	var hasChanges bool

	switch {
	case e.ComponentType.memcmp:
		e.checkChangesUsingSliceCompare(tick)

	case e.ComponentType.TriviallyHashable:
		memorySlices := e.ComponentType.MemorySlices
		for row := range n {
			hashValue := hashOf(memorySlices, e.ptrTo(row))
			if hashes[row] != hashValue {
				changed[row] = tick
				hashes[row] = hashValue
				hasChanges = true
			}
		}

	case e.ComponentType.Maphash != nil:
		maphash := e.ComponentType.Maphash

		for row := range n {
			hashValue := maphash(e.Get(row))
			if hashes[row] != hashValue {
				changed[row] = tick
				hashes[row] = hashValue
				hasChanges = true
			}
		}
	}

	if hasChanges {
		e.LastChanged = tick
	}
}

const noOffset = math.MaxUint64

func (e *ErasedColumn) checkChangesUsingSliceCompare(tick Tick) {
	// no need to check if we do not have any items
	itemSize := e.itemSize
	if itemSize == 0 {
		return
	}

	// view of the current data as a byte slice
	slice := (*(buf)(e.ptrTo(0)))[:uintptr(e.len)*e.itemSize]

	var hasChanges bool

	// keep track of the min and max offset of dirty bytes to copy over later
	var minOffset = uintptr(math.MaxInt)
	var maxOffset = uintptr(0)

	for item := uintptr(0); item < uintptr(e.len); item++ {
		// start comparing from the current item
		offset := sliceCompare(slice, e.sliceCopy, item*itemSize)
		if offset == noOffset {
			break
		}

		// calculate item of the first change
		item = offset / itemSize
		e.changed[item] = tick
		hasChanges = true

		minOffset = min(minOffset, offset)
		maxOffset = max(maxOffset, (item+1)*itemSize)
	}

	if hasChanges {
		e.LastChanged = tick
		copy(e.sliceCopy[minOffset:maxOffset], slice[minOffset:maxOffset])
	}

	if debug {
		if !bytes.Equal(slice, e.sliceCopy) {
			panic("more changes detected")
		}
	}
}

func (e *ErasedColumn) Len() int {
	return e.len
}

// Access creates a ColumnAccess that can be used to query Len columns.
// This method can also be called on a nil instance. The resulting ColumnAccess
// instance will return only nil pointers.
func (e *ErasedColumn) Access() ColumnAccess {
	if e == nil {
		return ColumnAccess{}
	}

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
