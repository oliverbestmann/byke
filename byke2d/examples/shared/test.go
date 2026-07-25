package shared

import (
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"maps"
	"os"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
)

var InTesting = os.Getenv("BYKE_RUN_OFFSCREEN_TEST") == "true"

type Hashes map[int]byke2d.Hash

func RunAppInTest(app byke.App, frameCount int, expected Hashes) {
	if !InTesting {
		app.MustRun()
		return
	}

	actual := Hashes{}

	var callback byke2d.DebugFrameCallback = func(frameIndex int, image *image.NRGBA) error {
		if _, ok := expected[frameIndex]; !ok {
			return nil
		}

		actual[frameIndex] = calculateImageHash(image)
		return saveImage(frameIndex, image)
	}

	app.AddPlugin(byke2d.PluginDebugDumpCamera(callback))

	// override window with fixed size offscreen window for testing
	app.InsertResource(byke2d.WindowConfig{
		Width:  640,
		Height: 480,
		Offscreen: &byke2d.OffscreenWindowConfig{
			FrameCount: frameCount,
		},
	})

	// force fallback adapter
	_ = os.Setenv("WGPU_FORCE_FALLBACK_ADAPTER", "1")

	app.MustRun()

	// compare hashes
	if !maps.Equal(actual, expected) {
		_, _ = fmt.Fprintf(os.Stderr, "Expected %#v\n", expected)
		_, _ = fmt.Fprintf(os.Stderr, "Got %#v\n", actual)
		os.Exit(1)
	}
}

func calculateImageHash(im *image.NRGBA) byke2d.Hash {
	mask := uint32(0b_11111111_11110000_11110000_11110000)

	var hash byke2d.Hash

	pixels := byke2d.ByteSliceAsValues[uint32](im.Pix)
	for idx := range pixels {
		hash.Int(pixels[idx] & mask)
	}

	return hash
}

func saveImage(frameIndex int, im *image.NRGBA) error {
	path := fmt.Sprintf("/tmp/image-%05d.png", frameIndex)
	slog.Info("Saving image", slog.String("path", path))

	fp, err := os.Create(path)
	if err != nil {
		return err
	}

	defer fp.Close()

	err = png.Encode(fp, im)
	if err != nil {
		return err
	}

	return nil
}
