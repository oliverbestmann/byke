package audio

import (
	"time"
)

type Sample interface {
	int16 | float32
}

type StreamConfig struct {
	SampleRate         int
	Channels           int
	ChannelSampleCount int
}

func (s StreamConfig) SampleCount() int {
	return s.ChannelSampleCount * s.Channels
}

func (s StreamConfig) Duration() time.Duration {
	return time.Duration(s.ChannelSampleCount) * time.Second / time.Duration(s.SampleRate)
}

type AudioStream[S Sample] interface {
	Read(samples []S) (int, error)

	// Seek seeks the file to a specific sample index, or by a sample offset
	Seek(offset int64, whence int) (int64, error)

	Config() StreamConfig
}
