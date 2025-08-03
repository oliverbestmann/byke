package bykebiten

import (
	"bytes"
	"fmt"
	"github.com/oliverbestmann/byke/bykebiten/audio"
	"io"
	"time"
)

var AutoDecodeThreshold = 20 * time.Second

type AutoDecodeMode = uint8

const (
	// AutoDecodeModeThreshold decodes audio files that have a duration less than AutoDecodeThreshold during load
	AutoDecodeModeThreshold = iota
	AutoDecodeModeAlways
	AutoDecodeModeNever
)

type LoadAudioSettings struct {
	// By default, a short audio file (less than AutoDecodeThreshold) will be fully decoded
	// during load, this way it can be played with super low overhead.
	AutoDecode AutoDecodeMode
}

func (l *LoadAudioSettings) IsLoadSettings() {}

type AudioLoader struct{}

func (a AudioLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	// open the stream once
	stream, err := audio.OpenStream(r)
	if err != nil {
		return nil, fmt.Errorf("create stream for: %w", err)
	}

	var settings LoadAudioSettings
	if ctx.Settings != nil {
		settings = *ctx.Settings.(*LoadAudioSettings)
	}

	if settings.AutoDecode != AutoDecodeModeNever {
		always := settings.AutoDecode == AutoDecodeModeAlways
		threshold := stream.Config().Duration() < AutoDecodeThreshold
		if always || threshold {
			return a.openAsSamples(stream)
		}
	}

	return a.openAsStream(r, err)
}

func (a AudioLoader) openAsSamples(stream audio.AudioStream[float32]) (*AudioSource, error) {
	// now decode the stream into samples
	samples, err := audio.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("decoding stream: %w", err)
	}

	config := stream.Config()

	factory := func() audio.AudioStream[float32] {
		return audio.FromSamples(config, samples)
	}

	return &AudioSource{factory: factory}, nil
}

func (a AudioLoader) openAsStream(r io.ReadSeekCloser, err error) (*AudioSource, error) {
	// seek back to the beginning of the file
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind asset: %w", err)
	}

	// this is now more tricky. We have a single open file which supports seeking,
	// but we might require multiple parallel readers, each calling Seek & Read on
	// different threads.
	//
	// We solve this issue by just caching the raw asset as bytes, then creating
	// individual readers for it
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("buffer asset: %w", err)
	}

	factory := func() audio.AudioStream[float32] {
		stream, err := audio.OpenStream(bytes.NewReader(buf))
		if err != nil {
			// there should not be any error at opening the stream already succeeded once
			panic(fmt.Errorf("open stream: %w", err))
		}

		return stream
	}

	return &AudioSource{factory: factory}, nil
}

func (a AudioLoader) Extensions() []string {
	return []string{".ogg", ".mp3", ".wav"}
}
