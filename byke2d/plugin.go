package byke2d

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[ClearColor]()

var (
	TransformSystems  = &byke.SystemSet{}
	VisibilitySystems = &byke.SystemSet{}
	AudioSystems      = &byke.SystemSet{}
	DeriveSprites     = &byke.SystemSet{}
)

var (
	PreRender  = byke.MakeScheduleId("PreRender")
	Render     = byke.MakeScheduleId("Render")
	PostRender = byke.MakeScheduleId("PostRender")
)

var (
	RenderPhaseExtract           = &byke.SystemSet{Name: "RenderPhaseExtract"}
	RenderPhaseQueue             = &byke.SystemSet{Name: "RenderPhaseQueue"}
	RenderPhaseSort              = &byke.SystemSet{Name: "RenderPhaseSort"}
	RenderPhasePrepare           = &byke.SystemSet{Name: "RenderPhasePrepare"}
	RenderPhasePrepareResources  = &byke.SystemSet{Name: "RenderPhasePrepareResources"}
	RenderPhasePrepareBindGroups = &byke.SystemSet{Name: "RenderPhasePrepareBindGroups"}
	RenderPhaseExecute           = &byke.SystemSet{Name: "RenderPhaseExecute"}
	RenderPhaseCleanup           = &byke.SystemSet{Name: "RenderPhaseCleanup"}
)

var (
	Core2dOpaque         = &byke.SystemSet{Name: "Core2dOpaque"}
	Core2dTransparent    = &byke.SystemSet{Name: "Core2dTransparent"}
	Core2dPostProcessing = &byke.SystemSet{Name: "Core2dPostProcessing"}
	Core2dBlit           = &byke.SystemSet{Name: "Core2dBlit"}
)

func RenderPlugin(app *byke.App) {
	assetFs, ok := byke.ResourceOf[AssetFS](app.World())
	if !ok {
		assetFs = &AssetFS{FS: os.DirFS("assets")}
	}

	app.AddMakeSystemParam(makeViewQuery)

	app.InsertResource(RenderContext{})
	app.InsertResource(DefaultWindowConfig())
	app.InsertResource(DefaultSurfaceConfig())
	app.InsertResource(TonemappingLutTextures{})
	app.InsertResource(surfaceConfigState{})

	app.AddPlugin(pluginShader)

	app.InsertResource(byke.InitFromWorld(PipelineCacheFromWorld))

	app.InsertResource(byke.InitFromWorld(TextureCacheFromWorld))

	app.AddPlugin(ComponentUniformsPlugin[bloomUniforms])
	app.AddPlugin(ComponentUniformsPlugin[ColorGrading])

	// input resources
	app.InsertResource(Keys{})
	app.InsertResource(MouseButtons{})
	app.InsertResource(MouseCursor{})
	app.InsertResource(MouseCursorDelta{})

	app.InsertResource(AudioContext{audioContext})
	app.InsertResource(GlobalVolume{Volume: 1.0})
	app.InsertResource(GlobalSpatialScale{Scale: glm.Vec3f{1, 1, 1}})

	app.InsertResource(makeAssets(app.World(), assetFs.FS,
		TextureLoader{},
		AudioLoader{},
		GLTFLoader{},
	))

	app.AddMessage(byke.MessageType[AppExit]())

	app.AddSystems(byke.First, updateMouseCursorSystem)

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleVisibilitySystem, propagateVisibilitySystem).
		Chain().
		InSet(VisibilitySystems))

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleTransformSystem, propagateTransformSystem).
		Chain().
		InSet(TransformSystems))

	app.AddSystems(byke.RenderMain, renderMainSystem)

	app.AddSystems(byke.PostUpdate, byke.
		System(createAudioSinkSystem, adjustSpatialAudioVolume, cleanupAudioSinkSystem).
		Chain().
		InSet(AudioSystems))

	app.AddSystems(PreRender, cacheTextSystem)

	app.AddSystems(Render, byke.
		System(renderTextSystem).
		InSet(RenderPhaseExtract))

	app.AddSystems(Render, byke.
		System(prepareBloomUniformsSystem).
		Chain().
		InSet(RenderPhasePrepare))

	app.AddSystems(Core2d, byke.
		System(applyBloomSystem, tonemappingSystem).
		Chain().
		InSet(Core2dPostProcessing))

	app.ConfigureSystemSets(byke.PostUpdate, TransformSystems)
	app.ConfigureSystemSets(byke.PostUpdate, VisibilitySystems)

	app.ConfigureSystemSets(Render, ChainSystemSets(
		RenderPhaseExtract,
		RenderPhaseQueue,
		RenderPhaseSort,
		RenderPhasePrepare,
		RenderPhasePrepareResources,
		RenderPhasePrepareBindGroups,
		RenderPhaseExecute,
		RenderPhaseCleanup,
	),
	)

	app.ConfigureSystemSets(Core2d,
		ChainSystemSets(Core2dOpaque, Core2dTransparent, Core2dPostProcessing, Core2dBlit))

	app.AddSystems(byke.Last, readAppExitEventsSystem)

	app.AddPlugin(pluginRenderPhases)
	app.AddPlugin(pluginCamera)
	app.AddPlugin(pluginSprite)

	app.AddPlugin(pluginLights)
	app.AddPlugin(pluginMesh)
	app.AddPlugin(pluginMesh2d)
	app.AddPlugin(pluginMesh3d)

	app.AddPlugin(pluginDebug)
	app.AddPlugin(pluginGltf)
	app.AddPlugin(pluginAnimations)

	app.RunWorld(runWorld)
}

func ChainSystemSets(first *byke.SystemSet, rest ...*byke.SystemSet) *byke.SystemSet {
	curr := first

	for _, set := range rest {
		set.After(curr)
		curr = set
	}

	return first
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

	wctx, err := newContext(win.SurfaceDescriptor())
	if err != nil {
		return fmt.Errorf("initialize wgpu: %w", err)
	}

	defer wctx.Release()

	dumpContextInfo(wctx)

	world.InsertResource(PrimaryWindow{window: win})

	renderContext := byke.RequireResourceOf[RenderContext](world)
	renderContext.init(world, wctx)

	err = win.Run(func(state vyn.UpdateInputState) error {
		return updateWorld(world, state)
	})

	// unwrap AppExit errors
	if exit, ok := errors.AsType[AppExit](err); ok {
		err = exit.error
	}

	return err
}

func dumpContextInfo(ctx *wgpuContext) {
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
	byke.ComparableComponent[ClearColor]
	Color
}

type surfaceConfigState struct {
	Width  uint32
	Height uint32
	Config SurfaceConfig
}

type currentSurfaceValues struct {
	Texture     *wgpu.Texture
	TextureView *wgpu.TextureView
	Size        glm.Vec2f
	Format      wgpu.TextureFormat
}

func updateWorld(world *byke.World, makeInputState vyn.UpdateInputState) error {
	defer puffin.NewScope("frame").End()

	ctx, _ := byke.ResourceOf[RenderContext](world)
	win, _ := byke.ResourceOf[PrimaryWindow](world)

	ctx.Metrics.reset()

	surfaceWidth, surfaceHeight := win.window.GetSize()
	ensureSurfaceConfigured(ctx, world, surfaceWidth, surfaceHeight)

	// update input state in world
	updateInputState(world, makeInputState)

	// update the game state by running all schedules
	world.RunSchedule(byke.Main)

	// handle app exit by error
	if exit, ok := byke.ResourceOf[appExitState](world); ok {
		return exit.Error
	}

	// slog.Info("Render metrics", slog.Any("metrics", ctx.Metrics))

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

	defer puffin.NewScope("surface.Configure").End()

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
			wgpu.TextureFormatBGRA8UnormSrgb,
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

func renderMainSystem(
	world *byke.World,
	ctx *RenderContext,
	textureCache *TextureCache,
) {
	// get the surface texture (the actual screen)
	surfaceTexture := func() wgpu.SurfaceTexture {
		defer puffin.NewScope("surface.GetCurrentTexture").End()
		return ctx.Surface.GetCurrentTexture()
	}()

	surface, ok := surfaceTexture.Get()
	defer surface.Release()
	if !ok {
		slog.Warn(
			"Failed to get a current surface texture",
			slog.String("status", surfaceTexture.Status.String()),
		)

		time.Sleep(16 * time.Millisecond)
		return
	}

	surfaceWidth := surface.GetWidth()
	surfaceHeight := surface.GetHeight()

	surfaceViewFormat := wgpu.TextureFormatBGRA8UnormSrgb

	// create a view we can render to
	surfaceTextureView := surface.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "Surface",
		Format:          surfaceViewFormat,
		MipLevelCount:   1,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})
	defer surfaceTextureView.Release()

	// store the target in the world for the renderer to access it
	world.InsertResource(currentSurfaceValues{
		Texture:     surface,
		TextureView: surfaceTextureView,
		Format:      surfaceViewFormat,
		Size:        glm.Vec2f{float32(surfaceWidth), float32(surfaceHeight)},
	})
	defer world.RemoveResource(reflect.TypeFor[currentSurfaceValues]())

	// remove unused textures
	textureCache.Reset()

	world.RunSchedule(PreRender)
	world.RunSchedule(Render)
	world.RunSchedule(PostRender)

	// present the current frame
	puffin.Scoped("surface.Present", func() any {
		ctx.Surface.Present()
		return nil
	})
}
