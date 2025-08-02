package audio

import (
	"fmt"
	"io"
	"unsafe"
)

func ToReadSeeker[S Sample](stream AudioStream[S]) io.ReadSeeker {
	if stream, ok := stream.(*readerAudioStream[S]); ok {
		return stream.reader
	}

	return &toReadSeeker[S]{stream: stream}
}

type toReadSeeker[S Sample] struct {
	stream AudioStream[S]
}

func (a *toReadSeeker[S]) Read(p []byte) (int, error) {
	sampleSize := int(unsafe.Sizeof(S(0)))

	samples := BytesAsSamples[S](p)

	// round down to a full sample
	sampleCount := len(p) - len(p)%sampleSize
	samples = samples[:sampleCount/sampleSize]

	n, err := a.stream.Read(samples)
	return n * sampleSize, err
}

func (a *toReadSeeker[S]) Seek(offset int64, whence int) (int64, error) {
	sampleSize := int64(unsafe.Sizeof(S(0)))

	if offset%sampleSize != 0 {
		return 0, fmt.Errorf("seek %d not aligned with sample size %d", offset, sampleSize)
	}

	n, err := a.stream.Seek(offset/sampleSize, whence)
	return n * sampleSize, err
}
