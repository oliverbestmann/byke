package audio

import (
	"bytes"
	"errors"
	"io"
)

func ReadAll[S Sample](stream AudioStream[S]) ([]S, error) {
	sampleCount := max(0, stream.Config().SampleCount())
	samples := make([]S, 0, sampleCount)

	var buf [4096]S

	for {
		n, err := stream.Read(buf[:])
		if n > 0 {
			samples = append(samples, buf[:]...)
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		if n == 0 {
			return samples, nil
		}
	}
}

func FromSamples[S Sample](config StreamConfig, samples []S) AudioStream[S] {
	buf := SamplesAsBytes(samples)

	return &readerAudioStream[S]{
		reader: bytes.NewReader(buf),
		config: config,
	}
}
