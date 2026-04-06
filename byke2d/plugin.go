package byke2d

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/vyn"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func Plugin(app *byke.App) {
	app.InsertResource(DefaultWindowConfig())
	app.InsertResource(DefaultSurfaceConfig())
	app.InsertResource(surfaceConfigState{})

	// input resources
	app.InsertResource(Keys{})
	app.InsertResource(MouseButtons{})
	app.InsertResource(MouseCursor{})
	app.InsertResource(MouseCursorDelta{})

	app.AddMessage(byke.MessageType[AppExit]())

	app.AddSystems(byke.First, updateMouseCursorSystem)

	app.AddSystems(byke.Last, readAppExitEventsSystem)

	app.RunWorld(runWorld)
}

type WindowConfig struct {
	Title         string
	Width         int
	Height        int
	DisableResize bool
}

func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		Title:  "Byke App",
		Width:  1280,
		Height: 720,
	}
}

func DefaultSurfaceConfig() SurfaceConfig {
	return SurfaceConfig{
		Format:      wgpu.TextureFormatBGRA8Unorm,
		PresentMode: wgpu.PresentModeFifo,
	}
}

type PrimaryWindow struct {
	window  vyn.Window
	context *wx.Context
}

type ScreenSize struct {
	glm.Vec2f
}

func (w PrimaryWindow) Context() *wx.Context {
	return w.context
}

func runWorld(world *byke.World) error {
	conf, _ := byke.ResourceOf[WindowConfig](world)

	title := getOr(conf.Title, "Byke App")
	width := getOr(conf.Width, 1280)
	height := getOr(conf.Height, 720)

	win, err := vyn.NewWindow(width, height, title, !conf.DisableResize)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}

	defer win.Terminate()

	ctx, err := wx.New(win.SurfaceDescriptor())
	if err != nil {
		return fmt.Errorf("initialize wgpu: %w", err)
	}

	defer ctx.Release()

	dumpContextInfo(ctx)

	world.InsertResource(PrimaryWindow{window: win, context: ctx})

	err = win.Run(func(state vyn.UpdateInputState) error {
		return updateWorld(ctx, win, world, state)
	})

	// unwrap AppExit errors
	if exit, ok := errors.AsType[AppExit](err); ok {
		err = exit.error
	}

	return err
}

func dumpContextInfo(ctx *wx.Context) {
	if runtime.GOOS != "js" {
		// print adapter info
		adapterInfo := ctx.Adapter.GetInfo()
		fmt.Printf("Using device: %s\n", adapterInfo.Device)
		fmt.Printf("Description:  %s\n", adapterInfo.Description)
		fmt.Printf("Backend:      %s\n", adapterInfo.BackendType)
		fmt.Printf("Vendor:       %s\n", adapterInfo.Vendor)
	}
}

type SurfaceConfig struct {
	// Format is the wgpu.TextureFormat to use for the windows wgpu.Surface
	Format wgpu.TextureFormat

	// PresentMode is the desired present mode for the windows wgpu.Surface.
	PresentMode wgpu.PresentMode
}

type InputState struct {
	state vyn.InputState
}

type surfaceConfigState struct {
	Width  uint32
	Height uint32
	Config SurfaceConfig
}

func updateWorld(ctx *wx.Context, win vyn.Window, world *byke.World, makeInputState vyn.UpdateInputState) error {
	surfaceWidth, surfaceHeight := win.GetSize()
	ensureSurfaceConfigured(ctx, world, surfaceWidth, surfaceHeight)

	// get the surface texture (the actual screen)
	surface := ctx.Surface.GetCurrentTexture()

	// The surface must only be released if it was not rendered to.
	// To skip releasing the surface, we can set it to nil later.
	defer func() {
		if surface != nil {
			surface.Release()
		}
	}()

	// update input state in world
	updateInputState(world, makeInputState)

	// update the game state by running all schedules
	world.RunSchedule(byke.Main)

	// present the current frame
	ctx.Surface.Present()

	// we do not need to release the surface texture if present was successful
	surface = nil

	// handle app exit by error
	if exit, ok := byke.ResourceOf[appExitState](world); ok {
		return exit.Error
	}

	return nil
}

func updateInputState(world *byke.World, makeInputState vyn.UpdateInputState) {
	inputState := makeInputState()

	// store state in world for mouse cursors to update
	world.InsertResource(InputState{inputState})

	keys, _ := byke.ResourceOf[Keys](world)
	keys.state = inputState.Keys

	buttons, _ := byke.ResourceOf[MouseButtons](world)
	buttons.state = inputState.Mouse
}

func ensureSurfaceConfigured(ctx *wx.Context, world *byke.World, surfaceWidth uint32, surfaceHeight uint32) {
	state, _ := byke.ResourceOf[surfaceConfigState](world)
	surfaceConfig, _ := byke.ResourceOf[SurfaceConfig](world)

	if state.Width == surfaceWidth || state.Height == surfaceHeight || state.Config == *surfaceConfig {
		return
	}

	slog.Debug("Configure surface",
		slog.Int("width", int(surfaceWidth)),
		slog.Int("height", int(surfaceHeight)),
	)

	ctx.Surface.Configure(ctx.Device, &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      surfaceConfig.Format,
		Width:       surfaceWidth,
		Height:      surfaceHeight,
		PresentMode: surfaceConfig.PresentMode,
		AlphaMode:   wgpu.CompositeAlphaModeOpaque,
		ViewFormats: []wgpu.TextureFormat{
			surfaceConfig.Format,
		},

		DesiredMaximumFrameLatency: 1,
	})

	// update state to match the current configuration
	state.Width = surfaceWidth
	state.Height = surfaceHeight
	state.Config = *surfaceConfig

	// expose screen size in variable
	screenSize := glm.Vec2f{float32(surfaceWidth), float32(surfaceHeight)}
	world.InsertResource(ScreenSize{screenSize})
}

func getOr[T comparable](value, fallback T) T {
	var tZero T
	if value == tZero {
		return fallback // reconfigure surface if needed
	}

	return value
}

func readAppExitEventsSystem(c *byke.Commands, events *byke.MessageReader[AppExit]) {
	for _, ev := range events.Read() {
		c.InsertResource(appExitState{Error: ev})
	}
}

type appExitState struct {
	Error error
}
