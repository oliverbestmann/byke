package byke2d

import (
	"math"
	"reflect"
	"unsafe"

	"golang.org/x/exp/constraints"
)

type Hash uint64

func HashFor[T any]() Hash {
	var h Hash = 0x6AFEC7DF3CEBAE4D

	ty := reflect.TypeFor[T]()

	// get the pointer to the interface type and use that to initialize the hash
	type eface struct{ _, val unsafe.Pointer }

	h.Int(uintptr((*eface)(unsafe.Pointer(&ty)).val))

	return h
}

func (h *Hash) Pointer[T any](value *T) {
	h.Update(uint64(uintptr(unsafe.Pointer(value))))
}

func (h *Hash) Float32(value float32) {
	h.Update(uint64(math.Float32bits(value)))
}

func (h *Hash) Float64(value float64) {
	h.Update(math.Float64bits(value))
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

func hashIntegerSlice[T constraints.Integer](values []T) Hash {
	// start with a "randomish" hash
	var h Hash = 0x44CA2D356ECF5645

	// hash the length in there
	h.Int(len(values))

	for _, value := range values {
		h.Int(value)
	}

	return h
}
