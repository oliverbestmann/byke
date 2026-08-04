//go:build js

package byke2d

import (
	"fmt"
	"image"
	"io"
	"log/slog"
	"syscall/js"
	"time"

	"github.com/oliverbestmann/webgpu/jsx"
)

func decodeImage(r io.Reader) (image.Image, error) {
	startTime := time.Now()
	defer func() { slog.Info("Loading image", slog.Duration("duration", time.Since(startTime))) }()

	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read buffer to memory: %w", err)
	}

	return LoadImage(buf)
}

func LoadImage(data []byte) (image.Image, error) {
	bufIn := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(bufIn, data)

	promise := loadImageAsync.Invoke(bufIn)

	// wait for loading to finish
	obj, ok := jsx.Await(promise)
	if !ok {
		return nil, js.Error{Value: obj}
	}

	width := obj.Get("width").Int()
	height := obj.Get("height").Int()

	rgba := &image.NRGBA{
		Pix:    make([]byte, width*height*4),
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	js.CopyBytesToGo(rgba.Pix, obj.Get("data"))

	return rgba, nil
}

var loadImageAsync = js.FuncOf(func(this js.Value, args []js.Value) any {
	// first arg is "bytes: Uint8Array"
	jsCode := `
		return (async function() {
			const blob = new Blob([bytes]);
			const bitmap = await createImageBitmap(blob);

			const canvas = new OffscreenCanvas(
				bitmap.width,
				bitmap.height,
			);

			const ctx = canvas.getContext("2d");
			ctx.drawImage(bitmap, 0, 0);

			// get RGBA image data
			const imageData = ctx.getImageData(
				0,
				0,
				bitmap.width,
				bitmap.height,
			);

			return {
				width: bitmap.width,
				height: bitmap.height,
				data: imageData.data,
			};
		})();
	`

	fn := js.Global().Get("Function").New("bytes", jsCode)
	return fn.Invoke(args[0])
})
