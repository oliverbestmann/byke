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
	app.InsertResource(ClearColor{wx.ColorSRGBA(0.2, 0.2, 0.3, 1.0)})

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
	window vyn.Window
}

type RenderContext struct {
	*wx.Context
}

type ScreenSize struct {
	glm.Vec2f
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

	world.InsertResource(PrimaryWindow{window: win})
	world.InsertResource(RenderContext{Context: ctx})

	err = win.Run(func(state vyn.UpdateInputState) error {
		return updateWorld(world, state)
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

type ClearColor struct {
	wx.Color
}

type surfaceConfigState struct {
	Width  uint32
	Height uint32
	Config SurfaceConfig
}

func updateWorld(world *byke.World, makeInputState vyn.UpdateInputState) error {
	ctx, _ := byke.ResourceOf[RenderContext](world)
	win, _ := byke.ResourceOf[PrimaryWindow](world)

	surfaceWidth, surfaceHeight := win.window.GetSize()
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

	// create a view we can render to
	textureView := surface.CreateView(nil)
	defer textureView.Release()

	// clear the texture
	clearColor, _ := byke.ResourceOf[ClearColor](world)
	clearTexture(ctx, textureView, clearColor.Color)

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

func ensureSurfaceConfigured(ctx *RenderContext, world *byke.World, surfaceWidth uint32, surfaceHeight uint32) {
	state, _ := byke.ResourceOf[surfaceConfigState](world)
	surfaceConfig, _ := byke.ResourceOf[SurfaceConfig](world)

	if state.Width == surfaceWidth && state.Height == surfaceHeight && state.Config == *surfaceConfig {
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

func clearTexture(ctx *RenderContext, texView *wgpu.TextureView, color wx.Color) {
	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "ClearTexture"})
	defer enc.Release()

	carr := color.ToWGPU()

	desc := &wgpu.RenderPassDescriptor{
		Label: "ClearUsingTexture",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View: texView,
				// ResolveTarget: resolveView,
				LoadOp:  wgpu.LoadOpClear,
				StoreOp: wgpu.StoreOpStore,
				ClearValue: wgpu.Color{
					R: float64(carr[0]),
					G: float64(carr[1]),
					B: float64(carr[2]),
					A: float64(carr[3]),
				},
			},
		},
	}

	enc.BeginRenderPass(desc).End()

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "ClearTexture"})
	defer buf.Release()

	ctx.Submit(buf)
}
