package radix_c

import "unsafe"

// #include <radixsort.h>
import "C"

// #include <radixsort.h>
import "C"

type Value struct {
	Key   float32
	Index uint32
}

func Sort(values, scratch []Value) {
	valuesC := (*C.value_t)(unsafe.Pointer(unsafe.SliceData(values)))
	bufC := (*C.value_t)(unsafe.Pointer(unsafe.SliceData(scratch)))

	C.radixsort_c(valuesC, bufC, C.uint32_t(len(values)))
}
