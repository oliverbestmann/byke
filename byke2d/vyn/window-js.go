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

	grab bool
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
		event := args[0]
		scale := win.deviceScale()

		if document.Get("pointerLockElement").Equal(win.canvas) {
			// pageX/pageY are frozen while locked, accumulate a virtual cursor like glfw does.
			x := win.input.Mouse.CursorX + float32(event.Get("movementX").Float()*scale)
			y := win.input.Mouse.CursorY + float32(event.Get("movementY").Float()*scale)
			win.input.Mouse.position(x, y)
			return nil
		}

		pageX := event.Get("pageX").Float() * scale
		pageY := event.Get("pageY").Float() * scale
		win.input.Mouse.position(float32(pageX), float32(pageY))
		return nil
	}))

	win.canvas.Call("addEventListener", "pointerdown", js.FuncOf(func(this js.Value, args []js.Value) any {
		// touches have no pointermove before pointerdown, move the cursor to where the touch starts
		if !document.Get("pointerLockElement").Equal(win.canvas) {
			event := args[0]
			scale := win.deviceScale()
			win.input.Mouse.position(float32(event.Get("pageX").Float()*scale), float32(event.Get("pageY").Float()*scale))
		}

		win.input.Mouse.press(MouseButton(0))

		// the browser drops the lock on Escape, so request it again on every click
		if win.grab && !document.Get("pointerLockElement").Equal(win.canvas) {
			slog.Info("Requesting pointer lock")
			win.canvas.Call("requestPointerLock")
		}

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

func (g *jsWindow) SetCursorGrab(grab bool) {
	g.grab = grab

	if !grab {
		js.Global().Get("document").Call("exitPointerLock")
	}
}
