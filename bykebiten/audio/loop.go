package audio

import (
	"errors"
	"io"
)

type loopAudioStream[S Sample] struct {
	stream AudioStream[S]
}

func Loop[S Sample](stream AudioStream[S]) AudioStream[S] {
	return &loopAudioStream[S]{stream}
}

func (l *loopAudioStream[S]) Read(samples []S) (int, error) {
	n, err := l.stream.Read(samples)
	switch {
	case errors.Is(err, io.EOF), n == 0 && len(samples) > 0:

		// try to seek back
		_, errSeek := l.stream.Seek(0, io.SeekStart)
		if errSeek != nil {
			return n, errors.Join(err, errSeek)
		}

		// try to read a second time
		n, err = l.stream.Read(samples)
	}

	return n, err
}

func (l *loopAudioStream[S]) Seek(offset int64, whence int) (int64, error) {
	sampleCount := int64(l.stream.Config().SampleCount())
	if sampleCount == 0 {
		return 0, errors.ErrUnsupported
	}

	return l.stream.Seek(offset%sampleCount, whence)
}

func (l *loopAudioStream[S]) Config() StreamConfig {
	conf := l.stream.Config()
	conf.ChannelSampleCount = 0
	return conf
}
