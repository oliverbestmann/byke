package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
)

type Camera struct {
	byke.Component[Camera]
	RenderTarget RenderTarget
	Order        int
}

func (c Camera) RequireComponents() []spoke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
	}
}

func (c Camera) ToScreenSpace(gt GlobalTransform) ebiten.GeoM {
	var tr ebiten.GeoM
	tr.Rotate(float64(gt.Rotation))
	tr.Scale(gt.Scale.X, gt.Scale.Y)
	tr.Translate(gt.Translation.X, gt.Translation.Y)
	return tr
}

type RenderTarget struct {
	byke.Component[RenderTarget]

	// if the image is nil, the default screen will be used
	Image *ebiten.Image
}
