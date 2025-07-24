package spoke

import "C"
import (
	"unsafe"
)

func memcmpSlow(lhs, rhs unsafe.Pointer, n int) int {
	for idx := 0; idx < n; idx++ {
		a := *(*byte)(unsafe.Add(lhs, idx))
		b := *(*byte)(unsafe.Add(rhs, idx))
		if a != b {
			return idx
		}
	}

	return -1
}

func memcmp(lhs, rhs []byte, offset int) int {
	if len(lhs) != len(rhs) {
		panic("slices have different sizes")
	}

	n := len(lhs)

	ptrA := unsafe.Pointer(unsafe.SliceData(lhs))
	ptrB := unsafe.Pointer(unsafe.SliceData(rhs))

	if uintptr(ptrA)%8 != 0 || uintptr(ptrB)%8 != 0 {
		panic("pointers are not aligned")
	}

	if offset%8 != 0 {
		res := memcmpSlow(unsafe.Add(ptrA, offset), unsafe.Add(ptrB, offset), 8-offset%8)
		if res != -1 {
			return offset + res
		}

		offset += 8 - offset%8
	}

	for idx := offset; idx < n; idx += 8 {
		a := unsafe.Add(ptrA, idx)
		b := unsafe.Add(ptrB, idx)

		if *(*uint64)(a) == *(*uint64)(b) {
			continue
		}

		return idx + memcmpSlow(a, b, 8)
	}

	return -1
}
