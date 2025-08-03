package bykebiten

import (
	"bytes"
	"fmt"
	"github.com/oliverbestmann/byke/bykebiten/audio"
	"io"
	"log/slog"
)

type AudioLoader struct{}

func (a AudioLoader) Load(ctx LoadContext, r io.Reader) (any, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read asset into memory: %w", err)
	}

	// try to parse the file once.
	stream, err := audio.OpenStream(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("create stream for: %w", err)
	}

	config := stream.Config()

	// now decode the stream into samples
	samples, err := audio.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("decoding stream: %w", err)
	}

	slog.Debug("Cached audio asset",
		slog.String("path", ctx.Path),
		slog.Int("sampleCount", len(samples)),
	)

	factory := func() audio.AudioStream[float32] {
		return audio.FromSamples(config, samples)
	}

	return &AudioSource{factory: factory}, nil
}

func (a AudioLoader) Extensions() []string {
	return []string{".ogg", ".mp3"}
}
