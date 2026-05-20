//go:build js

package vyn

import (
	"log/slog"
	"syscall/js"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type jsWindow struct {
	canvas js.Value
	input  InputState
	hidpi  bool
}

func NewWindow(width, height int, title string, resizable bool) (Window, error) {
	document := js.Global().Get("document")

	canvas := document.Call("createElement", "canvas")
	document.Get("body").Call("appendChild", canvas)
	document.Set("title", title)

	canvas.Set("style", "width:100vw; height:100vh")

	win := &jsWindow{
		canvas: canvas,
		hidpi:  true,
	}

	configureInput(document, win)

	return win, nil
}

func configureInput(document js.Value, win *jsWindow) {
	win.canvas.Call("addEventListener", "pointermove", js.FuncOf(func(this js.Value, args []js.Value) any {
		scale := win.deviceScale()
		pageX := args[0].Get("pageX").Float() * scale
		pageY := args[0].Get("pageY").Float() * scale
		win.input.Mouse.position(float32(pageX), float32(pageY))
		return nil
	}))

	win.canvas.Call("addEventListener", "pointerdown", js.FuncOf(func(this js.Value, args []js.Value) any {
		win.input.Mouse.press(MouseButton(0))
		return nil
	}))

	win.canvas.Call("addEventListener", "pointerup", js.FuncOf(func(this js.Value, args []js.Value) any {
		win.input.Mouse.release(MouseButton(0))
		return nil
	}))

	document.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		key, ok := keyOf(args[0])
		if ok {
			win.input.Keys.press(key)
		}

		return nil
	}))

	document.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) any {
		key, ok := keyOf(args[0])
		if ok {
			win.input.Keys.release(key)
		}

		return nil
	}))
}

func keyOf(event js.Value) (key Key, ok bool) {
	jsCode := event.Get("code").String()

	key, ok = jsToKey[jsCode]
	if !ok {
		key := event.Get("key").String()
		slog.Warn(
			"Unknown key code",
			slog.String("event.code", jsCode),
			slog.String("event.key", key),
		)
	}

	return
}

func (g *jsWindow) ShouldClose() bool {
	return false
}

func (g *jsWindow) GetSize() (uint32, uint32) {
	ratio := g.deviceScale()

	vv := js.Global().Get("visualViewport")
	width := vv.Get("width").Int()
	height := vv.Get("height").Int()
	return uint32(float64(width) * ratio), uint32(float64(height) * ratio)
}

func (g *jsWindow) deviceScale() float64 {
	if !g.hidpi {
		// do not look at the devicePixelRatio
		return 1.0
	}

	return js.Global().Get("devicePixelRatio").Float()
}

func (g *jsWindow) SurfaceDescriptor() *wgpu.SurfaceDescriptor {
	return &wgpu.SurfaceDescriptor{Canvas: g.canvas}
}

func (g *jsWindow) Terminate() {
	// do nothing
}

func (g *jsWindow) Run(render func(inputState UpdateInputState) error) error {
	var updateInputState UpdateInputState = func() InputState {
		return g.input
	}

	errCh := make(chan error, 1)

	renderOnce := func() bool {
		g.resizeCanvas()

		if err := render(updateInputState); err != nil {
			errCh <- err
			return false
		}

		g.input.nextTick()

		return true
	}

	var renderAndSchedule js.Func

	renderAndSchedule = js.FuncOf(func(this js.Value, args []js.Value) any {
		// we must not block in a FuncOf callback. spawn a go routine and call
		// requestAnimationFrame later from there
		go func() {
			if renderOnce() {
				js.Global().Call("requestAnimationFrame", renderAndSchedule)
			}
		}()

		return nil
	})

	defer renderAndSchedule.Release()

	// trigger the async render loop
	renderAndSchedule.Invoke()

	// block until we get an error
	return <-errCh
}

func (g *jsWindow) resizeCanvas() {
	vv := js.Global().Get("visualViewport")
	viewWidth := vv.Get("width").Float()
	viewHeight := vv.Get("height").Float()

	ratio := g.deviceScale()

	g.canvas.Set("width", viewWidth*ratio)
	g.canvas.Set("height", viewHeight*ratio)
}
