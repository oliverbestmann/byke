package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/examples/ecsbiten/color"
	"github.com/oliverbestmann/byke/gm"
	"slices"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Layer]()
var _ = byke.ValidateComponent[Size]()
var _ = byke.ValidateComponent[ColorTint]()
var _ = byke.ValidateComponent[Anchor]()

type Sprite struct {
	byke.ComparableComponent[Sprite]
	Image *ebiten.Image
}

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		Layer{},
		Transform{
			Scale: gm.VecOf(1.0, 1.0),
		},
		AnchorCenter,
		ColorTint{Color: color.White},
	}
}

type Size struct {
	byke.ComparableComponent[Size]
	gm.Vec
}

type Layer struct {
	byke.ComparableComponent[Layer]
	Z float64
}

type Anchor struct {
	byke.ComparableComponent[Anchor]
	gm.Vec
}

var AnchorCenter = Anchor{Vec: gm.Vec{X: 0.5, Y: 0.5}}

type ColorTint struct {
	byke.ComparableComponent[ColorTint]
	color.Color
}

type RenderTarget struct {
	*ebiten.Image
}

type renderSpritesValue struct {
	Sprite          Sprite
	GlobalTransform GlobalTransform
	Layer           Layer
	ColorTint       ColorTint
	Anchor          Anchor
	Size            byke.Option[Size]
}

type renderSpritesCache struct {
	sprites []renderSpritesValue
}

func renderSpritesSystem(
	screen RenderTarget,
	sprites byke.Query[renderSpritesValue],
	cache *byke.Local[renderSpritesCache],
) {
	// re-use the slice
	items := slices.AppendSeq(cache.Value.sprites[:0], sprites.Items())

	defer func() {
		clear(items)
		cache.Value.sprites = items[:0]
	}()

	slices.SortFunc(items, func(a, b renderSpritesValue) int {
		switch {
		case a.Layer.Z < b.Layer.Z:
			return -1

		case a.Layer.Z > b.Layer.Z:
			return 1

		default:
			return 0
		}
	})

	for _, item := range items {
		size, hasCustomSize := item.Size.Get()
		imageSize := ImageSizeOf(item.Sprite.Image)

		var op ebiten.DrawImageOptions

		// offset by anchor
		offset := imageSize.MulEach(item.Anchor.Vec)
		op.GeoM.Translate(-offset.X, -offset.Y)

		// get transformation
		tr := item.GlobalTransform

		if hasCustomSize {
			// apply custom size if available
			scale := size.DivEach(imageSize)
			op.GeoM.Scale(scale.X, scale.Y)
		}

		// apply custom size based on transform
		op.GeoM.Scale(tr.Scale.X, tr.Scale.Y)

		// apply rotation
		op.GeoM.Rotate(float64(tr.Rotation))

		// move to target position
		op.GeoM.Translate(tr.Translation.X, tr.Translation.Y)

		// apply color
		op.ColorScale.Scale(item.ColorTint.Float32Values())

		screen.DrawImage(item.Sprite.Image, &op)
	}
}

func ImageSizeOf(image *ebiten.Image) gm.Vec {
	return gm.Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}
