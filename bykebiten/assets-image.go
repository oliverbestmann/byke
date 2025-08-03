package bykebiten

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"io"
)

type ImageLoader struct{}

func (i ImageLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	return ebiten.NewImageFromImage(img), nil
}

func (i ImageLoader) Extensions() []string {
	return []string{".png", ".jpg", ".jpeg"}
}
