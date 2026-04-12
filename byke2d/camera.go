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
		renderLayerZero,
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

func (p OrthographicProjection) ScreenToNDC(screenSize glm.Vec2f) glm.Mat3f {
	return glm.IdentityMat3[float32]().
		Scale(2.0, -2.0).
		Translate(p.ViewportOrigin.XY()).
		Translate(-0.5, -0.5).
		Scale(screenSize.Scale(1.0 / p.Scale).Reciprocal().XY())
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
