package byke2d

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"slices"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/vyn"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var TransformSystems = &byke.SystemSet{}
var VisibilitySystems = &byke.SystemSet{}
var AudioSystems = &byke.SystemSet{}
var DeriveSprites = &byke.SystemSet{}

var RenderSystems = &byke.SystemSet{}
var RenderPostProcessSystems = &byke.SystemSet{}

var (
	PreRender  = byke.MakeScheduleId("PreRender")
	Render     = byke.MakeScheduleId("Render")
	PostRender = byke.MakeScheduleId("PostRender")
)

func RenderPlugin(app *byke.App) {
	app.AddMakeSystemParam(makePipelinesSystemParamState)
	app.AddMakeSystemParam(makeComponentUniformsSystemParamState)

	assetFs, ok := byke.ResourceOf[AssetFS](app.World())
	if !ok {
		assetFs = &AssetFS{FS: os.DirFS("assets")}
	}

	app.InsertResource(DefaultWindowConfig())
	app.InsertResource(DefaultSurfaceConfig())
	app.InsertResource(surfaceConfigState{})
	app.InsertResource(makePipelineCache())

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
	))

	app.AddMessage(byke.MessageType[AppExit]())

	app.AddSystems(byke.First, updateMouseCursorSystem)

	app.AddSystems(byke.PostUpdate, byke.
		System(renderTextSystem).
		InSet(DeriveSprites))

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleVisibilitySystem, propagateVisibilitySystem).
		Chain().
		InSet(VisibilitySystems))

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleTransformSystem, propagateTransformSystem).
		Chain().
		InSet(TransformSystems))

	app.AddSystems(byke.RenderMain, driveRenderScheduleSystem)

	app.AddSystems(PreRender,
		byke.System(createAudioSinkSystem, adjustSpatialAudioVolume, cleanupAudioSinkSystem).
			Chain().
			InSet(AudioSystems))

	app.AddSystems(Render,
		byke.System(renderSpriteSystem).Chain())

	app.AddSystems(Render, byke.
		System(applyBloomSystem, tonemappingSystem).
		Chain().
		InSet(RenderPostProcessSystems))

	// Adding new sprites must run before transform & visibility propagation
	app.ConfigureSystemSets(byke.PostUpdate, DeriveSprites.Before(TransformSystems))
	app.ConfigureSystemSets(byke.PostUpdate, DeriveSprites.Before(VisibilitySystems))

	app.ConfigureSystemSets(Render, RenderSystems.Before(RenderPostProcessSystems))

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
		Format:      wgpu.TextureFormatBGRA8UnormSrgb,
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

	ctx, err := wx.New(win.SurfaceDescriptor())
	if err != nil {
		return fmt.Errorf("initialize wgpu: %w", err)
	}

	defer ctx.Release()

	dumpContextInfo(ctx)

	world.InsertResource(PrimaryWindow{window: win})
	world.InsertResource(buildRenderContext(ctx))

	renderContext, _ := byke.ResourceOf[RenderContext](world)
	world.InsertResource(TextureCache{Context: renderContext})

	err = win.Run(func(state vyn.UpdateInputState) error {
		return updateWorld(world, state)
	})

	// unwrap AppExit errors
	if exit, ok := errors.AsType[AppExit](err); ok {
		err = exit.error
	}

	return err
}

func buildRenderContext(ctx *wx.Context) RenderContext {
	return RenderContext{
		Context:         ctx,
		MipmapGenerator: makeMipmapGenerator(ctx),
	}
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
	byke.ComparableComponent[ClearColor]
	wx.Color
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
}

func updateWorld(world *byke.World, makeInputState vyn.UpdateInputState) error {
	defer puffin.NewScope("frame").End()

	ctx, _ := byke.ResourceOf[RenderContext](world)
	win, _ := byke.ResourceOf[PrimaryWindow](world)

	ctx.Metrics.reset()

	surfaceWidth, surfaceHeight := win.window.GetSize()
	ensureSurfaceConfigured(ctx, world, surfaceWidth, surfaceHeight)

	// get the surface texture (the actual screen)
	surface := func() *wgpu.Texture {
		defer puffin.NewScope("surface.GetCurrentTexture").End()
		return ctx.Surface.GetCurrentTexture()
	}()

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
	surfaceTextureView := surface.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "Surface",
		Format:          surface.GetFormat(),
		Dimension:       wgpu.TextureViewDimension2D,
		BaseMipLevel:    0,
		MipLevelCount:   1,
		BaseArrayLayer:  0,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})
	defer surfaceTextureView.Release()

	// store the target in the world for the renderer to access it
	world.InsertResource(currentSurfaceValues{
		Texture:     surface,
		TextureView: surfaceTextureView,
		Size:        glm.Vec2f{float32(surfaceWidth), float32(surfaceHeight)},
	})

	// update the game state by running all schedules
	world.RunSchedule(byke.Main)

	// present the current frame
	puffin.Scoped("surface.Present", func() any {
		ctx.Surface.Present()
		return nil
	})

	// we do not need to release the surface texture if present was successful
	surface = nil

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

func clearTarget(ctx *RenderContext, viewTarget *ViewTarget) {
	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "ClearTexture"})
	defer enc.Release()

	desc := &wgpu.RenderPassDescriptor{
		Label: "ClearUsingTexture",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			viewTarget.Attachment(),
		},
	}

	enc.BeginRenderPass(desc).End()

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "ClearTexture"})
	defer buf.Release()

	ctx.Submit(buf)

	viewTarget.DiscardContent()
}

func clearTexture(ctx *RenderContext, texView, resolveView *wgpu.TextureView, color wx.Color) {
	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "ClearTexture"})
	defer enc.Release()

	carr := color.ToWGPU()

	desc := &wgpu.RenderPassDescriptor{
		Label: "ClearUsingTexture",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:          texView,
				ResolveTarget: resolveView,
				LoadOp:        wgpu.LoadOpClear,
				StoreOp:       wgpu.StoreOpStore,
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

func resolveTexture(ctx *RenderContext, view, resolveView *wgpu.TextureView) {
	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "ClearTexture"})
	defer enc.Release()

	desc := &wgpu.RenderPassDescriptor{
		Label: "ResolveTexture",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:          view,
				ResolveTarget: resolveView,
				LoadOp:        wgpu.LoadOpLoad,
				StoreOp:       wgpu.StoreOpStore,
			},
		},
	}

	enc.BeginRenderPass(desc).End()

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "ResolveTexture"})
	defer buf.Release()

	ctx.Submit(buf)
}

func driveRenderScheduleSystem(world *byke.World,
	ctx *RenderContext,
	textureCache *TextureCache,
	surfaceValues currentSurfaceValues,
	camerasQuery byke.Query[cameraQueryValues],
	blitPipelines Pipelines[blitConfig],
) {
	// TODO reuse allocation
	cameras := camerasQuery.AppendTo(nil)

	// sort cameras to render in ascending camera order
	slices.SortFunc(cameras, func(a, b cameraQueryValues) int {
		return b.Camera.Order - a.Camera.Order
	})

	// remove the camera value from the world after rendering
	defer world.RemoveResource(reflect.TypeFor[CurrentCamera]())

	textureCache.Reset()

	for _, camera := range cameras {
		puffin.Scoped("byke2d.RenderCamera", func() any {
			if camera.Camera.Inactive {
				return nil
			}

			viewTarget, hasViewTarget := buildCameraViewTarget(
				textureCache,
				surfaceValues,
				camera.RenderTarget,
				camera.ClearColor.Color,
				camera.HDR.Exists(),
				camera.MSAA.Exists(),
			)

			if !hasViewTarget {
				return nil
			}

			currentCamera := CurrentCamera{
				Entity:       camera.EntityId,
				Projection:   camera.Projection,
				Transform:    camera.Transform,
				RenderLayers: camera.RenderLayers,
				ViewTarget:   viewTarget,

				ColorGrading: toOptional(camera.ColorGrading),
				Tonemapping:  toOptional(camera.Tonemapping),
				DebandDither: toOptional(camera.DebandDither),
			}

			world.InsertResource(currentCamera)

			world.RunSchedule(PreRender)
			world.RunSchedule(Render)
			world.RunSchedule(PostRender)

			blit := blitConfig{
				TargetFormat: viewTarget.SurfaceTextureFormat,
			}

			// blit into the target texture
			blitTexture(ctx,
				blitPipelines.Specialize(blit),
				viewTarget.UnsampledTexture(),
				viewTarget.SurfaceTextureView,
			)

			if camera.RenderTarget.Texture != nil {
				camera.RenderTarget.Texture.Updated(ctx)
			}

			return nil
		})
	}
}

func toOptional[T byke.IsComponent[T]](o byke.Option[T]) Optional[T] {
	value, ok := o.Get()
	return Optional[T]{Value: value, IsSet: ok}
}

type CurrentCamera struct {
	Entity       byke.EntityId
	Projection   OrthographicProjection
	Transform    GlobalTransform
	RenderLayers RenderLayers
	ViewTarget   *ViewTarget

	ColorGrading Optional[ColorGrading]
	Tonemapping  Optional[Tonemapping]
	DebandDither Optional[DebandDither]
}

type cameraQueryValues struct {
	EntityId     byke.EntityId
	Camera       Camera
	Projection   OrthographicProjection
	Transform    GlobalTransform
	RenderLayers RenderLayers
	RenderTarget RenderTarget
	ClearColor   ClearColor

	MSAA byke.Has[MSAA]

	HDR          byke.Has[HDR]
	ColorGrading byke.Option[ColorGrading]
	Tonemapping  byke.Option[Tonemapping]
	DebandDither byke.Option[DebandDither]
}
