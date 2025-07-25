package spoke

import (
	"unsafe"
)

func sliceCompareSlow(lhs, rhs unsafe.Pointer, n uintptr) uintptr {
	for idx := uintptr(0); idx < n; idx++ {
		a := *(*byte)(unsafe.Add(lhs, idx))
		b := *(*byte)(unsafe.Add(rhs, idx))
		if a != b {
			return idx
		}
	}

	return noOffset
}

func sliceCompare(lhs, rhs []byte, offset uintptr) uintptr {
	if len(lhs) != len(rhs) {
		panic("slices have different sizes")
	}

	n := uintptr(len(lhs))

	if n == 0 {
		// no changes if we have no bytes
		return noOffset
	}

	ptrA := unsafe.Pointer(unsafe.SliceData(lhs))
	ptrB := unsafe.Pointer(unsafe.SliceData(rhs))

	// validate pointers are aligned
	if !isAligned(uintptr(ptrA)) || !isAligned(uintptr(ptrB)) {
		panic("pointers are not aligned")
	}

	// offset might not be aligned, use slow comparison until
	// we're aligned again
	if !isAligned(offset) {
		// number of bytes to go until we're aligned again
		rem := 8 - offset%8

		res := sliceCompareSlow(unsafe.Add(ptrA, offset), unsafe.Add(ptrB, offset), rem)
		if res != noOffset {
			return offset + res
		}

		offset += rem
	}

	// offset must be aligned now
	if !isAligned(offset) {
		panic("offset not aligned")
	}

	for ; offset < n; offset += 8 {
		ptrA := unsafe.Add(ptrA, offset)
		ptrB := unsafe.Add(ptrB, offset)

		if *(*uint64)(ptrA) == *(*uint64)(ptrB) {
			continue
		}

		return offset + sliceCompareSlow(ptrA, ptrB, 8)
	}

	if offset < n {
		// some more bytes are available
		rem := n - offset
		if rem >= 8 {
			panic("should be less than 8 bytes")
		}

		ptrA := unsafe.Add(ptrA, offset)
		ptrB := unsafe.Add(ptrB, offset)
		return offset + sliceCompareSlow(ptrA, ptrB, rem)
	}

	return noOffset
}

func isAligned(ptr uintptr) bool {
	return ptr%8 == 0
}
