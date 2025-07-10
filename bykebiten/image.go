package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke/gm"
)

func imageSizeOf(image *ebiten.Image) gm.Vec {
	return gm.Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}
