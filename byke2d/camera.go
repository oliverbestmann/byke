package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
)

var _ = byke.ValidateComponent[Camera]()
var _ = byke.ValidateComponent[OrthographicProjection]()

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
		MsaaOff,
		ClearColor{
			Color: wx.ColorSRGBA(0.2, 0.2, 0.3, 1.0),
		},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeWindowSize{},
			Scale:          1,
		},
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

func (s ScalingModeAutoMin) ViewportSize(width, height float32) glm.Vec2f {
	// Compare Pixels of current width and minimal height and Pixels of minimal width with current height.
	// Then use bigger (min_height when true) as what it refers to (height when true) and calculate rest so it can't get under minimum.
	if width*s.MinHeight > s.MinWidth*height {
		return glm.Vec2f{width * s.MinHeight / height, s.MinHeight}
	} else {
		return glm.Vec2f{s.MinWidth, height * s.MinWidth / width}
	}
}

// ScalingModeAutoMax keeps the aspect ratio while the axes can’t be bigger than given maximum.
type ScalingModeAutoMax struct {
	MaxWidth, MaxHeight float32
}

func (s ScalingModeAutoMax) ViewportSize(width, height float32) glm.Vec2f {
	// Compare Pixels of current width and maximal height and Pixels of maximal width with current height.
	// Then use smaller (max_height when true) as what it refers to (height when true) and calculate rest so it can't get over maximum.
	if width*s.MaxHeight < s.MaxWidth*height {
		return glm.Vec2f{width * s.MaxHeight / height, s.MaxHeight}
	} else {
		return glm.Vec2f{s.MaxWidth, height * s.MaxWidth / width}
	}
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
func (v *ViewValues) SurfaceToNDC() glm.Mat3f {
	return glm.Mat3f{}.
		Scale(2.0, 2.0).
		Translate(-0.5, -0.5)
}

// CameraToSurface maps a value from Camera space to a Surface space. Surface
// space is described by pixel coordinates with origin at 0 in the lower left corner.
func (v *ViewValues) CameraToSurface() glm.Mat3f {
	viewportSize := v.Projection.ScalingMode.ViewportSize(v.SurfaceSize.XY())

	return glm.Mat3f{}.
		Translate(v.Projection.ViewportOrigin.XY()).
		Scale(v.Projection.Scale, v.Projection.Scale).
		Scale(viewportSize.Reciprocal().XY())
}

// WorldToCamera maps a point from World space into Camera space.
// This just applies the Cameras position. It does not apply the
// cameras projection.
func (v *ViewValues) WorldToCamera() glm.Mat3f {
	t := v.Transform

	return glm.RotationMat3[float32](t.Rotation).
		Scale(t.Scale.XY()).
		Translate(t.Translation.Scale(-1).XY())
}
