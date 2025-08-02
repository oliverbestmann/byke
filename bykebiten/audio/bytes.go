package audio

import "unsafe"

func SamplesAsBytes[S int16 | float32](samples []S) []byte {
	ptr := (*byte)(unsafe.Pointer(unsafe.SliceData(samples)))
	return unsafe.Slice(ptr, unsafe.Sizeof(S(0))*uintptr(len(samples)))
}

func BytesAsSamples[S int16 | float32](buf []byte) []S {
	ptr := unsafe.Pointer(unsafe.SliceData(buf))

	if uintptr(ptr)%unsafe.Alignof(S(0)) != 0 {
		panic("buffer not aligned with sample type")
	}

	return unsafe.Slice((*S)(ptr), uintptr(len(buf))/unsafe.Sizeof(S(0)))
}
