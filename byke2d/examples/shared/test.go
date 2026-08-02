package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"maps"
	"os"
	"slices"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
)

var InTesting = os.Getenv("BYKE_RUN_OFFSCREEN_TEST") == "true"
var WriteSnapshots = os.Getenv("BYKE_WRITE_SNAPSHOTS") == "true"

type Snapshots map[int]byke2d.Hash

type FramesToSnapshot []int

func RunAppInTest(app byke.App, framesToSnapshot FramesToSnapshot) {
	if !InTesting {
		app.MustRun()
		return
	}

	// load existing snapshots
	expected := loadSnapshots()

	frameCount := slices.Max(framesToSnapshot) + 15

	actual := Snapshots{}

	var callback byke2d.DebugFrameCallback = func(frameIndex int, image *image.NRGBA) error {
		if !slices.Contains(framesToSnapshot, frameIndex) {
			return nil
		}

		// record snapshot for this frame
		actual[frameIndex] = calculateSnapshot(image)
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

	// if we had no snapshots, write them to the file
	if len(expected) == 0 && len(actual) > 0 || WriteSnapshots {
		writeSnapshots(actual)
		return
	}

	// compare snapshots
	if !maps.Equal(actual, expected) {
		_, _ = fmt.Fprintf(os.Stderr, "Expected %#v\n", expected)
		_, _ = fmt.Fprintf(os.Stderr, "Got %#v\n", actual)
		os.Exit(1)
	}
}

func writeSnapshots(snapshots Snapshots) {
	encoded, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		panic(err)
	}

	path := "snapshots.json"
	err = os.WriteFile(path, encoded, 0644)
	if err != nil {
		panic(err)
	}
}

func loadSnapshots() Snapshots {
	path := "snapshots.json"
	slog.Info("Loading snapshots", slog.String("path", path))

	fp, err := os.Open(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}

		slog.Warn("Hashes not found", slog.String("path", path))
		return nil
	}

	defer fp.Close()

	var snapshots Snapshots
	if err := json.NewDecoder(fp).Decode(&snapshots); err != nil {
		panic(err)
	}

	return snapshots
}

func calculateSnapshot(im *image.NRGBA) byke2d.Hash {
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
