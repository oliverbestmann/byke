//go:build cgo

package radix

import "unsafe"

// #include <radixsort.h>
import "C"

func doSort(values, scratch []Value) {
	valuesC := (*C.value_t)(unsafe.Pointer(unsafe.SliceData(values)))
	bufC := (*C.value_t)(unsafe.Pointer(unsafe.SliceData(scratch)))

	C.radixsort_c(valuesC, bufC, C.uint32_t(len(values)))
}
