package ktx2

import (
	"embed"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

//go:embed *.ktx2
var files embed.FS

func TestParseHeader(t *testing.T) {
	fp, _ := files.Open("pisa_specular_rgb9e5_zstd.ktx2")
	reader := fp.(io.ReadSeeker)

	h, err := Open(reader)
	require.NoError(t, err)

	faces, err := h.Faces()
	require.NoError(t, err)

	fmt.Println(h.TextureType())

	for _, face := range faces {
		continue
		im := &image.NRGBA{
			Pix:    face.Buffer,
			Stride: int(face.Width * 4),
			Rect:   image.Rect(0, 0, int(face.Width), int(face.Height)),
		}

		fp, err := os.Create(fmt.Sprintf("out-level=%d-face=%d.png", face.Level, face.Face))
		require.NoError(t, err)

		err = png.Encode(fp, im)
		require.NoError(t, err)

		fmt.Println(face)
	}

	spew.Dump(h)
}
