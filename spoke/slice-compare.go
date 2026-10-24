package spoke

import (
	"bytes"
	"math"
)

const noOffset = math.MaxUint64

// sliceCompare returns the offset of the first byte at or after offset where
// lhs and rhs differ, or noOffset if they are equal from there on.
func sliceCompare(lhs, rhs []byte, offset uintptr) uintptr {
	if len(lhs) != len(rhs) {
		panic("slices have different sizes")
	}

	const chunk = 64

	for start := int(offset); start < len(lhs); start += chunk {
		end := min(start+chunk, len(lhs))
		if bytes.Equal(lhs[start:end], rhs[start:end]) {
			continue
		}

		for idx := start; idx < end; idx++ {
			if lhs[idx] != rhs[idx] {
				return uintptr(idx)
			}
		}
	}

	return noOffset
}
