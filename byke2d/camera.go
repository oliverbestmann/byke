package byke2d

import (
	"math/rand/v2"
	"reflect"
	"sort"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var (
	_ = byke.ValidateComponent[Camera]()
	_ = byke.ValidateComponent[OrthographicProjection]()
)

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
		System(prepareGlobals).
		InSet(RenderPhasePrepareResources))

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
		},

		ViewUniforms{},
		BinnedRenderPhase[Opaque]{},
		SortableRenderPhase[Transparent]{},
	}
}

type OrthographicProjection struct {
	byke.Component[OrthographicProjection]

	// Origin of the camera. Set this to (0.5, 0.5) to center the Camera.
	ViewportOrigin glm.Vec2f

	ScalingMode ScalingMode

	// Distance of the near and far plane in camera direction.
	// If both values are set to zero, 0 and 1 is assumed.
	Near, Far float32
}

func (o OrthographicProjection) ToMat4f(viewSize glm.Vec2f) glm.Mat4f {
	viewSize = o.ScalingMode.ViewportSize(viewSize.XY())

	offset := o.ViewportOrigin.Mul(viewSize)
	left, right := -offset[0], -offset[0]+viewSize[0]
	top, bottom := -offset[1], -offset[1]+viewSize[1]

	near, far := o.nearFar()

	rcpWidth := 1.0 / viewSize[0]
	rcpHeight := 1.0 / viewSize[1]
	r := 1.0 / (far - near)

	return glm.Mat4f{
		{2 * rcpWidth, 0, 0, 0},
		{0, 2 * rcpHeight, 0, 0},
		{0, 0, r, 0},
		{-(left + right) * rcpWidth, -(top + bottom) * rcpHeight, -r * near, 1.0},
	}
}

func (o OrthographicProjection) nearFar() (near float32, far float32) {
	if o.Near == 0 && o.Far == 0 {
		return 0, 1
	}

	return o.Near, o.Far
}

type PerspectiveProjection struct {
	byke.Component[PerspectiveProjection]

	Fov  glm.Rad
	Near float32
	Far  float32
}

func (p PerspectiveProjection) ToMat4f(viewSize glm.Vec2f) glm.Mat4f {
	aspect := viewSize[0] / viewSize[1]
	return glm.Perspective(p.Fov, aspect, p.Near, p.Far)
}

var DefaultPerspectiveProjection = PerspectiveProjection{
	Fov:  glm.DegToRad(70),
	Near: 0.1,
	Far:  1000,
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

type Projection interface {
	ToMat4f(viewSize glm.Vec2f) glm.Mat4f
}

type ViewValues struct {
	// Camera transformation
	CameraTransform GlobalTransform

	// Camera projection
	Projection Projection

	// Surface size
	SurfaceSize glm.Vec2f
	WorldToClip glm.Mat4f
}

// SurfaceToNDC maps from Surface pixel coordinates to NDC (normalized device coordinates).
// NDC is from -1 to +1 on both axis.
func (v *ViewValues) SurfaceToNDC() glm.Mat4f {
	return glm.IdentityMat4f()
	// return glm.ScaleMat4f(2.0, 2.0, 1.0).
	// 	Translate(-0.5, -0.5, 0)
}

// WorldToCamera maps a point from World space into Camera space.
// This just applies the Cameras position. It does not apply the
// cameras projection.
func (v *ViewValues) WorldToCamera() glm.Mat4f {
	inv, ok := inverseAeffine(v.CameraTransform.Affine)
	if !ok {
		panic("not invertable")
	}

	return inv
}

func prepareViewUniformsSystem(
	vt byke.VirtualTime,
	viewsQuery byke.Query[struct {
		_                      byke.With[Camera]
		EntityId               byke.EntityId
		Transform              GlobalTransform
		OrthographicProjection byke.Option[OrthographicProjection]
		PerspectiveProjection  byke.Option[PerspectiveProjection]
		ViewTarget             *ViewTarget
		ViewUniforms           *ViewUniforms
		TAAA                   byke.Has[TAA]
	}],
) {
	for view := range viewsQuery.Items() {
		persp, perspOk := view.PerspectiveProjection.Get()
		ortho, orthoOk := view.OrthographicProjection.Get()

		var projection Projection
		switch {
		case perspOk:
			projection = persp

		case orthoOk:
			projection = ortho

		default:
			continue
		}

		cameraToClip := projection.ToMat4f(view.ViewTarget.Size)

		vv := ViewValues{
			CameraTransform: view.Transform,
			WorldToClip:     cameraToClip,
		}

		if view.TAAA.Exists() {
			offset := taaaOffsets[vt.Frames%4]
			vv.WorldToClip.TranslateAssign(offset[0], offset[1], 0)
		}

		*view.ViewUniforms = ViewUniforms{
			ScreenToNDC:   vv.SurfaceToNDC(),
			WorldToScreen: vv.WorldToClip.Mul(vv.WorldToCamera()),
		}
	}
}

type ViewBindGroup struct {
	BindGroup     *wgpu.BindGroup
	BufferViews   *wgpu.Buffer
	BufferGlobals *wgpu.Buffer
}

var ViewBindGroupLayout = SequentialLayoutWithLabel(
	"ViewUniforms",
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false),
)

func prepareGlobals(ctx *RenderContext, vt *byke.VirtualTime, g *ViewBindGroup) {
	var w wgsl.StructWriter
	w.AppendFloat32(float32(vt.Elapsed.Seconds()))
	w.AppendFloat32(vt.DeltaSecs)
	w.AppendUint(uint32(vt.Frames))
	w.AppendUint(rand.Uint32())
	w.WriteTo(ctx, &g.BufferGlobals, "Globals", wgpu.BufferUsageUniform)
}

func createViewUniformsBindGroup(
	ctx *RenderContext,
	viewBindGroup *ViewBindGroup,
	viewUniforms *ComponentUniforms[ViewUniforms],
) {
	bindingView := viewUniforms.Binding()
	if bindingView.Buffer != viewBindGroup.BufferViews {
		viewBindGroup.BindGroup.Release()
	}

	viewBindGroup.BindGroup = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "View Uniforms",
		Layout: ctx.CreateBindGroupLayout(ViewBindGroupLayout),
		Entries: Sequential(
			bindingView,
			BindingBuffer(viewBindGroup.BufferGlobals),
		),
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
			commands.Entity(camera.EntityId).Remove[ViewTarget]()
			continue
		}

		commands.Entity(camera.EntityId).Insert(viewTarget)

		width, height := uint32(viewTarget.Size[0]), uint32(viewTarget.Size[1])
		depthTexture := buildCameraViewDepthTexture(textureCache, glm.Vec2u{width, height}, camera.MSAA.Exists())
		commands.Entity(camera.EntityId).Insert(depthTexture)
	}
}

func blitCameraToTargetSystem(
	ctx *RenderContext,
	pipelines *PipelineCache,

	viewsQuery ViewQuery[struct {
		Camera       Camera
		ClearColor   ClearColor
		ViewTarget   *ViewTarget
		RenderTarget *RenderTarget
	}],
) {
	view := viewsQuery.Get()

	blit := blitConfig{
		Format:     view.ViewTarget.SurfaceTextureFormat,
		AlphaBlend: view.ClearColor.Alpha() < 1,
	}

	// blit into the target texture
	blitTextureSimple(
		ctx,
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

func inverseAeffine(m glm.Mat4f) (glm.Mat4f, bool) {
	// Convert to row-major variables for readability:
	a := m[0][0]
	b := m[1][0]
	c := m[2][0]

	d := m[0][1]
	e := m[1][1]
	f := m[2][1]

	g := m[0][2]
	h := m[1][2]
	i := m[2][2]

	det :=
		a*(e*i-f*h) -
			b*(d*i-f*g) +
			c*(d*h-e*g)

	if det == 0 {
		return glm.Mat4f{}, false
	}

	invDet := float32(1.0) / det

	var inv glm.Mat4f

	// Adjugate / determinant
	inv[0][0] = (e*i - f*h) * invDet
	inv[1][0] = (c*h - b*i) * invDet
	inv[2][0] = (b*f - c*e) * invDet

	inv[0][1] = (f*g - d*i) * invDet
	inv[1][1] = (a*i - c*g) * invDet
	inv[2][1] = (c*d - a*f) * invDet

	inv[0][2] = (d*h - e*g) * invDet
	inv[1][2] = (b*g - a*h) * invDet
	inv[2][2] = (a*e - b*d) * invDet

	t := m[3]

	// tInv = -(inv * t)
	inv[3] = glm.Vec4f{
		-(inv[0][0]*t[0] + inv[1][0]*t[1] + inv[2][0]*t[2]),
		-(inv[0][1]*t[0] + inv[1][1]*t[1] + inv[2][1]*t[2]),
		-(inv[0][2]*t[0] + inv[1][2]*t[1] + inv[2][2]*t[2]),
		1,
	}

	return inv, true
}
