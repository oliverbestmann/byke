package byke2d

import (
	"errors"
	"fmt"
	"image"
	"io"

	_ "image/jpeg"
	_ "image/png"

	"github.com/oliverbestmann/byke"
)

type LoadTextureSettings struct {
	Sampler      SamplerConfig
	LinearColors bool
}

func (*LoadTextureSettings) IsLoadSettings() {}

type TextureLoader struct{}

func (i TextureLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	var settings LoadTextureSettings
	if ctx.Settings != nil {
		settings = *ctx.Settings.(*LoadTextureSettings)
	}

	renderContext, ok := byke.ResourceOf[RenderContext](ctx.World)
	if !ok {
		return nil, errors.New("no RenderContext in world")
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	return NewTextureFromImage(renderContext, img, settings.Sampler, !settings.LinearColors), nil
}

func (i TextureLoader) Extensions() []string {
	return []string{".png", ".jpg", ".jpeg"}
}
