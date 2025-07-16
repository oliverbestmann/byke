package arch

import (
	"hash/maphash"
	"reflect"
	"unsafe"
)

type Row uint32

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
		byteSlice := (*buf(start))[:slice.Len]
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
		panic("memorySlicesOf only works with struct types")
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

func typeIsTriviallyHashable(t reflect.Type) bool {
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
			if !typeIsTriviallyHashable(t.Field(idx).Type) {
				return false
			}
		}

		return true

	default:
		return false
	}
}

func typeHasPointers(t reflect.Type) bool {
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
		reflect.Complex128:

		return false

	case reflect.Array:
		return typeHasPointers(t.Elem())

	case reflect.Struct:
		for idx := range t.NumField() {
			if typeHasPointers(t.Field(idx).Type) {
				return true
			}
		}

		return false

	default:
		return true
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
