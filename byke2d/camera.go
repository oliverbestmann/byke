package byke2d

import (
	"reflect"
	"sort"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Camera]()
var _ = byke.ValidateComponent[OrthographicProjection]()

func pluginCamera(app *byke.App) {
	app.AddPlugin(ComponentUniformsPlugin[ViewUniforms])

	app.InsertResource(ViewBindGroup{})

	app.AddSystems(Render, byke.
		System(prepareViewUniformsSystem).
		InSet(RenderPhasePrepare))

	app.AddSystems(Render, byke.
		System(updateCameraViewTargetSystem).
		InSet(RenderPhasePrepare))

	app.AddSystems(Render, byke.
		System(createViewUniformsBindGroup).
		InSet(RenderPhasePrepareBindGroups))

	app.AddSystems(Render, byke.
		System(driveCameraSchedules).
		InSet(RenderPhaseExecute))

	app.AddSystems(Core2d, byke.
		System(blitCameraToTargetSystem).
		InSet(Core2dBlit))
}

type Camera struct {
	byke.Component[Camera]

	// Inactive marks the camera as not active - it will not render.
	Inactive bool

	// SubCameraView holds an optional sub rectangle of the cameras render target to render to.
	// The rectangle is given relative to the render targets full size, so it is provided as
	// values between 0 and 1.
	// SubCameraView *glm.Rect

	// Cameras are rendered sorted by ascending order value
	Order int
}

func (Camera) RequireComponents() []spoke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
		PrimaryWindowRenderTarget,
		renderLayerZero,
		ClearColor{
			Color: ColorSRGBA(0.2, 0.2, 0.3, 1.0),
		},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeWindowSize{},
			Scale:          1,
		},

		ViewUniforms{},
		RenderPhase{},
	}
}

type OrthographicProjection struct {
	byke.Component[OrthographicProjection]
	// Origin of the camera. Set this to (0.5, 0.5) to center the Camera.
	ViewportOrigin glm.Vec2f

	ScalingMode ScalingMode

	// Extra scale to multiply on top of the ScalingMode. Can be used for zooming.
	Scale float32
}

type ScalingMode interface {
	ViewportSize(width, height float32) glm.Vec2f
}

type ScalingModeWindowSize struct{}

func (s ScalingModeWindowSize) ViewportSize(width, height float32) glm.Vec2f {
	return glm.Vec2f{width, height}
}

type ScalingModeFixed struct {
	Viewport glm.Vec2f
}

func (s ScalingModeFixed) ViewportSize(width, height float32) glm.Vec2f {
	return s.Viewport
}

// ScalingModeAutoMin keeps the aspect ratio while the axes can’t be smaller than given minimum.
type ScalingModeAutoMin struct {
	MinWidth, MinHeight float32
}

func (s ScalingModeAutoMin) ViewportSize(width, height float32) (viewportSize glm.Vec2f) {
	// Compare Pixels of current width and minimal height and Pixels of minimal width with current height.
	// Then use bigger (min_height when true) as what it refers to (height when true) and calculate rest so it can't get under minimum.
	if width*s.MinHeight > s.MinWidth*height {
		viewportSize = glm.Vec2f{width * s.MinHeight / height, s.MinHeight}
	} else {
		viewportSize = glm.Vec2f{s.MinWidth, height * s.MinWidth / width}
	}

	return
}

// ScalingModeAutoMax keeps the aspect ratio while the axes can’t be bigger than given maximum.
type ScalingModeAutoMax struct {
	MaxWidth, MaxHeight float32
}

func (s ScalingModeAutoMax) ViewportSize(width, height float32) (viewportSize glm.Vec2f) {
	// Compare Pixels of current width and maximal height and Pixels of maximal width with current height.
	// Then use smaller (max_height when true) as what it refers to (height when true) and calculate rest so it can't get over maximum.
	if width*s.MaxHeight < s.MaxWidth*height {
		viewportSize = glm.Vec2f{width * s.MaxHeight / height, s.MaxHeight}
	} else {
		viewportSize = glm.Vec2f{s.MaxWidth, height * s.MaxWidth / width}
	}

	return
}

type ScalingModeFixedVertical struct {
	ViewportHeight float32
}

func (s ScalingModeFixedVertical) ViewportSize(width, height float32) glm.Vec2f {
	return glm.Vec2f{width * s.ViewportHeight / height, s.ViewportHeight}
}

type ScalingModeFixedHorizontal struct {
	ViewportWidth float32
}

func (s ScalingModeFixedHorizontal) ViewportSize(width, height float32) glm.Vec2f {
	return glm.Vec2f{s.ViewportWidth, height * s.ViewportWidth / width}
}

type ViewValues struct {
	// Camera transformation
	Transform GlobalTransform

	// Camera projection
	Projection OrthographicProjection

	// Surface size
	SurfaceSize glm.Vec2f
}

// SurfaceToNDC maps from Surface pixel coordinates to NDC (normalized device coordinates).
// NDC is from -1 to +1 on both axis.
func (v *ViewValues) SurfaceToNDC() glm.Mat4f {
	return glm.ScaleMat4f(2.0, 2.0, 1.0).
		Translate(-0.5, -0.5, 0)
}

// CameraToSurface maps a value from Camera space to a Surface space. Surface
// space is described by pixel coordinates with origin at 0 in the lower left corner.
func (v *ViewValues) CameraToSurface() glm.Mat4f {
	viewportSize := v.Projection.ScalingMode.ViewportSize(v.SurfaceSize.XY())

	return glm.IdentityMat4f().
		Translate(v.Projection.ViewportOrigin.Extend(1.0).XYZ()).
		Scale(v.Projection.Scale, v.Projection.Scale, 1).
		Scale(viewportSize.Reciprocal().Extend(1.0).XYZ())
}

// WorldToCamera maps a point from World space into Camera space.
// This just applies the Cameras position. It does not apply the
// cameras projection.
func (v *ViewValues) WorldToCamera() glm.Mat4f {
	return v.Transform.Affine
}

func prepareViewUniformsSystem(
	vt byke.VirtualTime,
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		EntityId     byke.EntityId
		Transform    GlobalTransform
		Projection   OrthographicProjection
		ViewTarget   *ViewTarget
		ViewUniforms *ViewUniforms
		TAAA         byke.Has[TAA]
	}],
) {
	for view := range viewsQuery.Items() {
		vv := ViewValues{
			Transform:   view.Transform,
			Projection:  view.Projection,
			SurfaceSize: view.ViewTarget.Size,
		}

		cameraToSurface := vv.CameraToSurface()

		if view.TAAA.Exists() {
			offset := taaaOffsets[vt.Frames%4]
			cameraToSurface.TranslateAssign(offset[0], offset[1], 0)
		}

		*view.ViewUniforms = ViewUniforms{
			ScreenToNDC:   vv.SurfaceToNDC(),
			WorldToScreen: cameraToSurface.Mul(vv.WorldToCamera()),
		}
	}
}

type ViewBindGroup struct {
	BindGroup *wgpu.BindGroup
	buffer    *wgpu.Buffer
}

var ViewBindGroupLayout = SequentialLayoutWithLabel("ViewUniforms",
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
)

func createViewUniformsBindGroup(
	ctx *RenderContext,
	pipelineCache *PipelineCache,
	viewBindGroup *ViewBindGroup,
	viewUniforms *ComponentUniforms[ViewUniforms],
) {
	binding := viewUniforms.Binding()
	if binding.Buffer != viewBindGroup.buffer {
		viewBindGroup.BindGroup.Release()
	}

	viewBindGroup.BindGroup = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   "View Uniforms",
		Layout:  pipelineCache.BindGroupLayout(ViewBindGroupLayout),
		Entries: Sequential(binding),
	})
}

func updateCameraViewTargetSystem(
	commands *byke.Commands,
	textureCache *TextureCache,
	surfaceValues currentSurfaceValues,
	camerasQuery byke.Query[struct {
		EntityId     byke.EntityId
		RenderTarget RenderTarget
		ClearColor   ClearColor
		MSAA         byke.Has[MSAA]
		HDR          byke.Has[HDR]
	}],
) {
	for camera := range camerasQuery.Items() {
		viewTarget, hasViewTarget := buildCameraViewTarget(
			textureCache,
			surfaceValues,
			camera.RenderTarget,
			camera.ClearColor.Color,
			camera.HDR.Exists(),
			camera.MSAA.Exists(),
		)

		if !hasViewTarget {
			commands.Entity(camera.EntityId).Update(byke.RemoveComponent[ViewTarget]())
			continue
		}

		commands.Entity(camera.EntityId).Insert(viewTarget)
	}
}

func blitCameraToTargetSystem(
	ctx *RenderContext,
	pipelines PipelineCache,

	viewsQuery ViewQuery[struct {
		Camera       Camera
		ViewTarget   *ViewTarget
		RenderTarget *RenderTarget
	}],
) {
	view := viewsQuery.Get()

	blit := blitConfig{
		Format: view.ViewTarget.SurfaceTextureFormat,
	}

	// blit into the target texture
	blitTextureSimple(ctx,
		pipelines.Specialize(blit),
		view.ViewTarget.UnsampledTexture(),
		view.ViewTarget.SurfaceTextureView,
	)

	if view.RenderTarget.Texture != nil {
		view.RenderTarget.Texture.Updated(ctx)
	}
}

func driveCameraSchedules(
	world *byke.World,
	camerasQuery byke.Query[struct {
		EntityId byke.EntityId
		Camera   Camera
	}],
) {
	// TODO reuse allocation
	cameras := camerasQuery.AppendTo(nil)

	// sort cameras to render in ascending camera order
	sort.Slice(cameras, func(a, b int) bool {
		return cameras[a].Camera.Order < cameras[b].Camera.Order
	})

	defer world.RemoveResource(reflect.TypeFor[CurrentView]())

	for _, camera := range cameras {
		world.InsertResource(CurrentView(camera.EntityId))
		world.RunSchedule(Core2d)
	}
}
