package byke2d

import (
	"errors"
	"fmt"
	"image"
	"io"
	"path"
	"reflect"

	_ "image/jpeg"
	_ "image/png"
)

type LoadTextureSettings struct {
	TextureFromImageOptions
}

func (*LoadTextureSettings) IsLoadSettings() {}

type TextureLoader struct{}

func (i TextureLoader) Type() reflect.Type {
	return reflect.TypeFor[*Texture]()
}

func (i TextureLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	var settings LoadTextureSettings
	if ctx.Settings != nil {
		settings = *ctx.Settings.(*LoadTextureSettings)
	}

	if settings.Label == "" {
		settings.Label = path.Base(ctx.Path)
	}

	renderContext, ok := ctx.World.ResourceOf[RenderContext]()
	if !ok {
		return nil, errors.New("no RenderContext in world")
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	if settings.Label == "" {
		// use the path as label
		settings.Label = ctx.Path
	}

	return NewTextureFromImage(renderContext, img, settings.TextureFromImageOptions), nil
}

func (i TextureLoader) Extensions() []string {
	return []string{".png", ".jpg", ".jpeg"}
}
