//go:build !js

package vyn

import (
	"fmt"
	"log/slog"

	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpu"
	"github.com/oliverbestmann/webgpu/wgpuglfw"
)

type glfwWindow struct {
	win   *glfw.Window
	input InputState
}

func NewWindow(width, height int, title string, resizable bool) (Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("initialize glfw: %w", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)

	if resizable {
		glfw.WindowHint(glfw.Resizable, glfw.True)
	} else {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	}

	window, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create window: %w", err)
	}

	w := &glfwWindow{
		win: window,
	}

	configureInput(window, &w.input)

	return w, nil
}

func (g *glfwWindow) ShouldClose() bool {
	return g.win.ShouldClose()
}

func (g *glfwWindow) GetSize() (uint32, uint32) {
	width, height := g.win.GetFramebufferSize()
	return uint32(width), uint32(height)
}

func (g *glfwWindow) SurfaceDescriptor() *wgpu.SurfaceDescriptor {
	return wgpuglfw.GetSurfaceDescriptor(g.win)
}

func (g *glfwWindow) Terminate() {
	g.win.Destroy()
	glfw.Terminate()
}

func (g *glfwWindow) Run(render func(input UpdateInputState) error) error {
	var updateInputState UpdateInputState = func() InputState {
		g.input.nextTick()
		glfw.PollEvents()
		return g.input
	}

	for !g.win.ShouldClose() {
		if err := render(updateInputState); err != nil {
			return err
		}
	}

	return nil
}

func configureInput(window *glfw.Window, input *InputState) {
	window.SetKeyCallback(func(_win *glfw.Window, glfwKey glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Repeat {
			return
		}

		key, ok := keyOf(glfwKey)
		if !ok {
			return
		}

		switch action {
		case glfw.Press:
			input.Keys.press(key)

		case glfw.Release:
			input.Keys.release(key)
		}
	})

	window.SetMouseButtonCallback(func(_win *glfw.Window, btn glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		button := MouseButton(btn)

		switch action {
		case glfw.Press:
			input.Mouse.press(button)
		case glfw.Release:
			input.Mouse.release(button)
		}
	})

	window.SetCursorPosCallback(func(win *glfw.Window, xpos float64, ypos float64) {
		scaleX, scaleY := win.GetContentScale()
		input.Mouse.position(float32(xpos)*scaleX, float32(ypos)*scaleY)
	})
}

func keyOf(glfwKey glfw.Key) (key Key, ok bool) {
	key, ok = glfwToKey[glfwKey]
	if !ok {
		slog.Warn(
			"Unknown key code",
			slog.String("key", glfw.GetKeyName(glfwKey, 0)),
		)
	}

	return
}
