package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
)

type Camera struct {
	byke.Component[Camera]

	// Target holds the texture to render this camera to.
	Target RenderTarget

	// Inactive marks the camera as not active - it will not render.
	Inactive bool

	// SubCameraView holds an optional sub rectangle of the cameras render target to render to.
	// The rectangle is given relative to the render targets full size, so it is provided as
	// values between 0 and 1.
	SubCameraView *gm.Rect

	// The clear color. If defined, the camera will clear its render region first.
	// If set to nil, the camera will simply render on top of anything in the viewport.
	//
	// If the render target is the screen itself, the screen is always cleared.
	ClearColor *color.Color

	// Cameras are rendered by ascending order value
	Order int
}

func (c Camera) RequireComponents() []spoke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
		OrthographicProjection{
			ViewportOrigin: gm.Vec{X: 0.5, Y: 0.5},
			ScalingMode:    ScalingModeWindowSize{},
			Scale:          1,
		},
	}
}

type RenderTarget struct {
	byke.Component[RenderTarget]

	// if the image is nil, the default screen will be used
	Image *ebiten.Image
}

type OrthographicProjection struct {
	byke.Component[OrthographicProjection]
	// Origin of the camera. Set this to (0.5, 0.5) to center the Camera.
	ViewportOrigin gm.Vec

	ScalingMode ScalingMode

	// Extra scale to multiply on top of the ScalingMode. Can be used for zooming.
	Scale float64
}

type ScalingMode interface {
	ViewportSize(width, height float64) gm.Vec
}

type ScalingModeWindowSize struct{}

func (s ScalingModeWindowSize) ViewportSize(width, height float64) gm.Vec {
	return gm.VecOf(width, height)
}

type ScalingModeFixed struct {
	Viewport gm.Vec
}

func (s ScalingModeFixed) ViewportSize(width, height float64) gm.Vec {
	return s.Viewport
}

// ScalingModeAutoMin keeps the aspect ratio while the axes can’t be smaller than given minimum.
type ScalingModeAutoMin struct {
	MinWidth, MinHeight float64
}

func (s ScalingModeAutoMin) ViewportSize(width, height float64) gm.Vec {
	// Compare Pixels of current width and minimal height and Pixels of minimal width with current height.
	// Then use bigger (min_height when true) as what it refers to (height when true) and calculate rest so it can't get under minimum.
	if width*s.MinHeight > s.MinWidth*height {
		return gm.VecOf(width*s.MinHeight/height, s.MinHeight)
	} else {
		return gm.VecOf(s.MinWidth, height*s.MinWidth/width)
	}
}

// ScalingModeAutoMax keeps the aspect ratio while the axes can’t be bigger than given maximum.
type ScalingModeAutoMax struct {
	MaxWidth, MaxHeight float64
}

func (s ScalingModeAutoMax) ViewportSize(width, height float64) gm.Vec {
	// Compare Pixels of current width and maximal height and Pixels of maximal width with current height.
	// Then use smaller (max_height when true) as what it refers to (height when true) and calculate rest so it can't get over maximum.
	if width*s.MaxHeight < s.MaxWidth*height {
		return gm.VecOf(width*s.MaxHeight/height, s.MaxHeight)
	} else {
		return gm.VecOf(s.MaxWidth, height*s.MaxWidth/width)
	}
}

type ScalingModeFixedVertical struct {
	ViewportHeight float64
}

func (s ScalingModeFixedVertical) ViewportSize(width, height float64) gm.Vec {
	return gm.VecOf(width*s.ViewportHeight/height, s.ViewportHeight)
}

type ScalingModeFixedHorizontal struct {
	ViewportWidth float64
}

func (s ScalingModeFixedHorizontal) ViewportSize(width, height float64) gm.Vec {
	return gm.VecOf(s.ViewportWidth, height*s.ViewportWidth/width)
}
