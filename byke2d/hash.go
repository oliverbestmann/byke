package byke2d

import (
	"math"
	"unsafe"

	"golang.org/x/exp/constraints"
)

type Hash uint64

func (h *Hash) Pointer[T any](value *T) {
	h.Update(uint64(uintptr(unsafe.Pointer(value))))
}

func (h *Hash) Float32(value float32) {
	h.Update(uint64(math.Float32bits(value)))
}

func (h *Hash) Int[T constraints.Integer](value T) {
	h.Update(uint64(value))
}

func (h *Hash) Bool(value bool) {
	if value {
		h.Update(1)
	} else {
		h.Update(2)
	}
}

func (h *Hash) Update(u uint64) {
	*h = splitMix64(uint64(*h) + u)
}

func splitMix64(x uint64) Hash {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return Hash(x)
}
