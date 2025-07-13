package arch

import (
	"hash/maphash"
	"reflect"
	"unsafe"
)

type Row uint32

type RowGetter = func(Row) ComponentValue

type Column interface {
	Append(tick Tick, component ErasedComponent)
	Copy(from, to Row)
	Truncate(n Row)
	Get(row Row) ComponentValue
	Update(tick Tick, row Row, cv ErasedComponent)
	Import(other Column, row Row)
	CheckChanged(tick Tick)
	Len() int

	Getter() func(Row) ComponentValue
}

type TypedColumn[C IsComponent[C]] struct {
	ComponentType *ComponentType
	Values        []TypedComponentValue[C]
}

type ComparableTypedColumn[C IsComparableComponent[C]] struct {
	TypedColumn[C]
	memorySlices      []memorySlice
	triviallyHashable bool
}

func MakeColumnOf[C IsComponent[C]](componentType *ComponentType) MakeColumn {
	if componentType.Type.Size() == 0 {
		// a zero sized type can never be modified in place, we can just go
		// with an immutable column.
		return MakeImmutableColumnOf[C](componentType)
	}

	return func() Column {
		return &TypedColumn[C]{
			ComponentType: componentType,
		}
	}
}

func MakeComparableColumnOf[C IsComparableComponent[C]](componentType *ComponentType) MakeColumn {
	if componentType.Type.Size() == 0 {
		// a zero sized type can never be modified in place, we can just go
		// with an immutable column.
		return MakeImmutableColumnOf[C](componentType)
	}

	reflectType := reflect.TypeFor[C]()
	triviallyHashable := isTriviallyHashable(reflectType)

	var memorySlices []memorySlice
	if triviallyHashable {
		memorySlices = memorySlicesOf(reflectType, 0, nil)
	}

	return func() Column {
		return &ComparableTypedColumn[C]{
			TypedColumn: TypedColumn[C]{
				ComponentType: componentType,
			},

			triviallyHashable: triviallyHashable,
			memorySlices:      memorySlices,
		}
	}
}

func (c *TypedColumn[C]) Len() int {
	return len(c.Values)
}

func (c *TypedColumn[C]) Append(tick Tick, component ErasedComponent) {
	value := any(component).(*C)

	c.Values = append(c.Values, TypedComponentValue[C]{
		Value:   *value,
		Added:   tick,
		Changed: tick,
	})
}

func (c *ComparableTypedColumn[C]) Append(tick Tick, component ErasedComponent) {
	value := any(component).(*C)

	c.Values = append(c.Values, TypedComponentValue[C]{
		Value:   *value,
		Added:   tick,
		Changed: tick,
		Hash:    c.hashOf(value),
	})
}

func (c *ComparableTypedColumn[C]) hashOf(value *C) HashValue {
	if c.triviallyHashable {
		return hashOf(c.memorySlices, unsafe.Pointer(value))
	}

	// use normal maphash
	return maphashOf(value)
}

func (c *TypedColumn[C]) Copy(from, to Row) {
	c.Values[to] = c.Values[from]
}

func (c *TypedColumn[C]) Import(other Column, row Row) {
	otherColumn := other.(*TypedColumn[C])
	c.Values = append(c.Values, otherColumn.Values[row])
}

func (c *ComparableTypedColumn[C]) Import(other Column, row Row) {
	otherColumn := other.(*ComparableTypedColumn[C])
	c.Values = append(c.Values, otherColumn.Values[row])
}

func (c *TypedColumn[C]) Truncate(n Row) {
	clear(c.Values[n:])
	c.Values = c.Values[:n]
}

func (c *TypedColumn[C]) Get(row Row) ComponentValue {
	t := &c.Values[row]

	return ComponentValue{
		Added:   t.Added,
		Changed: t.Changed,
		Value:   any(&t.Value).(ErasedComponent),
	}
}

func (c *TypedColumn[C]) Getter() func(Row) ComponentValue {
	return c.Get
}

func (c *TypedColumn[C]) Update(tick Tick, row Row, erasedValue ErasedComponent) {
	target := &c.Values[row]
	target.Value = *any(erasedValue).(*C)
	target.Changed = tick
}

func (c *ComparableTypedColumn[C]) Update(tick Tick, row Row, erasedValue ErasedComponent) {
	target := &c.Values[row]
	target.Value = erasedValue.(C)

	if hash := c.hashOf(&target.Value); hash != target.Hash {
		target.Hash = hash
		target.Changed = tick
	}
}

func (c *TypedColumn[C]) CheckChanged(tick Tick) {
	panic("not supported")
}

func (c *ComparableTypedColumn[C]) CheckChanged(tick Tick) {
	for idx := range c.Values {
		cv := &c.Values[idx]

		if hash := c.hashOf(&cv.Value); hash != cv.Hash {
			cv.Hash = hash
			cv.Changed = tick
		}
	}
}

// maphashOf is a safe hash that uses the maphash package to hash a value of type C.
func maphashOf[C IsComparableComponent[C]](value *C) HashValue {
	return HashValue(maphash.Comparable[C](seed, *value))
}

// hashOf calculates the hash of a value. This method is not as safe as maphashOf, but a lot faster.
// This will hash the memorySlice values passed in.
func hashOf(memorySlices []memorySlice, value unsafe.Pointer) HashValue {
	var hashValue HashValue

	//goland:noinspection GoRedundantConversion
	for _, slice := range memorySlices {
		start := unsafe.Add(value, slice.Start)
		byteSlice := unsafe.Slice((*byte)(start), slice.Len)

		hashValue = hashValue ^ HashValue(maphash.Bytes(seed, byteSlice))
	}

	return hashValue
}

type memorySlice struct {
	Start uintptr
	Len   uintptr
}

// memorySlicesOf returns a slice of memorySlice instances that define the bytes that
// are actually defined and do not contain padding within the type. The type itself must
// be a comparable struct.
func memorySlicesOf(t reflect.Type, base uintptr, slices []memorySlice) []memorySlice {
	if t.Kind() != reflect.Struct {
		panic("memorySlicesOf only works with comparable struct types")
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		fieldStart := base + field.Offset

		// Recursively check embedded structs (anonymous or not)
		if field.Type.Kind() == reflect.Struct {
			slices = memorySlicesOf(field.Type, fieldStart, slices)
			continue
		}

		if len(slices) > 0 {
			prev := &slices[len(slices)-1]
			if prev.Start+prev.Len == fieldStart {
				// we join the previous field, extend it
				prev.Len += field.Type.Size()
				continue
			}
		}

		// there was a gap, add another slice
		slices = append(slices, memorySlice{
			Start: fieldStart,
			Len:   field.Type.Size(),
		})
	}

	return slices
}

func isTriviallyHashable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Pointer,
		reflect.UnsafePointer:

		return true

	case reflect.Struct:
		for idx := range t.NumField() {
			if !isTriviallyHashable(t.Field(idx).Type) {
				return false
			}
		}

		return true

	default:
		return false
	}
}

func typeHasPaddingBytes(t reflect.Type) bool {
	slices := memorySlicesOf(t, 0, nil)
	if len(slices) == 0 {
		return false
	}

	last := slices[len(slices)-1]
	size := last.Start + last.Len
	return size < t.Size()
}
