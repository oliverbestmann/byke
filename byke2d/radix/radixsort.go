package radix

import "math"

//go:inline
func floatToSortableU32(f float32) uint32 {
	x := math.Float32bits(f)
	mask := uint32(int32(x)>>31) | 0x80000000
	return x ^ mask
}

func radixsortGo(values []Value, scratch []Value) {
	src := values
	dst := scratch

	const RADIX = 256

	var count [RADIX]uint32

	for pass := 0; pass < 4; pass++ {
		clear(count[:])

		shift := uint(pass * 8)

		/* Histogram */
		for i := uint32(0); i < uint32(len(src)); i++ {
			v := src[i] // bounds-checked, but usually eliminated
			k := floatToSortableU32(v.Key)
			count[(k>>shift)&0xFF]++
		}

		/* Prefix sum */
		sum := uint32(0)
		for i := 0; i < RADIX; i++ {
			c := count[i]
			count[i] = sum
			sum += c
		}

		/* Scatter */
		for i := uint32(0); i < uint32(len(src)); i++ {
			v := src[i]
			k := floatToSortableU32(v.Key)

			idx := (k >> shift) & 0xFF

			// local alias helps compiler avoid repeated bounds checks
			d := dst
			c := count[idx]
			d[c] = v
			count[idx] = c + 1
		}

		// swap buffers
		src, dst = dst, src
	}

	if &src[0] != &values[0] {
		panic("invariant: even number of sort passes must lead to src == values")
	}
}
