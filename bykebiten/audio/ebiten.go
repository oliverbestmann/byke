package audio

import (
	"errors"
	"io"
	"unsafe"
)

type ebitenAudioStream interface {
	io.ReadSeeker
	Length() int64
	SampleRate() int
}

type readerAudioStream[S Sample] struct {
	reader io.ReadSeeker
	config StreamConfig
}

func AdaptAudioStream[S Sample](r ebitenAudioStream) AudioStream[S] {
	st := &readerAudioStream[S]{
		reader: r,
		config: StreamConfig{
			SampleRate: r.SampleRate(),
			Channels:   2,
		},
	}

	sampleSize := int(unsafe.Sizeof(S(0)))

	byteCount := int(r.Length())
	if byteCount > 0 {
		st.config.ChannelSampleCount = byteCount * sampleSize / st.config.Channels
	}

	return st
}

func (r *readerAudioStream[S]) Read(samples []S) (int, error) {
	buf := SamplesAsBytes(samples)

	n, err := io.ReadFull(r.reader, buf)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		// this is fine
		err = nil
	}

	sampleSize := int(unsafe.Sizeof(S(0)))
	return max(0, n) / sampleSize, err
}

func (r *readerAudioStream[S]) Seek(offset int64, whence int) (int64, error) {
	sampleSize := int64(unsafe.Sizeof(S(0)))

	n, err := r.reader.Seek(offset*sampleSize, whence)
	return n / sampleSize, err
}

func (r *readerAudioStream[S]) Config() StreamConfig {
	return r.config
}
