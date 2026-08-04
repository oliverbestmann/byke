package byke2d

import (
	"fmt"
	"image"
	"io"
	"reflect"
)

type ImageLoader struct{}

func (i ImageLoader) Type() reflect.Type {
	return reflect.TypeFor[image.Image]()
}

func (i ImageLoader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	img, err := decodeImage(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	return img, nil
}

func (i ImageLoader) Extensions() []string {
	return []string{".png", ".jpg", ".jpeg"}
}
