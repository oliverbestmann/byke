//go:build !js

package byke2d

import (
	"image"
	"io"

	_ "image/jpeg"
	_ "image/png"
)

func decodeImage(r io.Reader) (image.Image, error) {
	image, _, err := image.Decode(r)
	return image, err
}
