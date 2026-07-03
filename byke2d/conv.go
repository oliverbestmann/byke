package byke2d

import (
	"fmt"
	"unsafe"
)

func ValuesAsByteSlice[T any](values []T) []byte {
	var tZero T

	ptr := unsafe.SliceData(values)
	bytes := (*byte)(unsafe.Pointer(ptr))
	return unsafe.Slice(bytes, uintptr(len(values))*unsafe.Sizeof(tZero))
}

func ValueAsByteSlice[T any](value T) []byte {
	bytes := (*byte)(unsafe.Pointer(&value))
	return unsafe.Slice(bytes, unsafe.Sizeof(value))
}

func ByteSliceAsValues[T any](bytes []byte) []T {
	var tZero T

	// TODO check alignment & size

	ptr := unsafe.SliceData(bytes)
	values := (*T)(unsafe.Pointer(ptr))
	return unsafe.Slice(values, uintptr(len(bytes))/unsafe.Sizeof(tZero))
}

func ValuesToValues[A, B any](values []A) []B {
	var aZero A
	var bZero B

	if unsafe.Sizeof(aZero) != unsafe.Sizeof(bZero) {
		panic(fmt.Errorf("type %T and %T have different sizes", aZero, bZero))
	}

	ptr := unsafe.SliceData(values)
	bValues := (*B)(unsafe.Pointer(ptr))
	return unsafe.Slice(bValues, len(values))
}
