package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	ecs "gobevy"
	"gobevy/examples/ecsbiten/color"
	"slices"
)

var _ = ecs.ValidateComponent[Transform]()
var _ = ecs.ValidateComponent[Sprite]()
var _ = ecs.ValidateComponent[Layer]()
var _ = ecs.ValidateComponent[Size]()
var _ = ecs.ValidateComponent[ColorTint]()
var _ = ecs.ValidateComponent[Anchor]()

type Transform struct {
	ecs.Component[Transform]
	Translation Vec
	Scale       Vec
	Rotation    Rad
}

type Rad float64

type Sprite struct {
	ecs.Component[Sprite]
	Image *ebiten.Image
}

func (Sprite) RequireComponents() []ecs.AnyComponent {
	return []ecs.AnyComponent{
		Layer{},
		Transform{
			Scale: VecOf(1.0, 1.0),
		},
		AnchorCenter,
		ColorTint{Color: color.White},
	}
}

type Size struct {
	ecs.Component[Size]
	Vec
}

type Layer struct {
	ecs.Component[Layer]
	Z float64
}

type Anchor struct {
	ecs.Component[Anchor]
	Vec
}

var AnchorCenter = Anchor{Vec: Vec{X: 0.5, Y: 0.5}}

type ColorTint struct {
	ecs.Component[ColorTint]
	color.Color
}

type RenderTarget struct {
	*ebiten.Image
}

type RenderSpritesValue struct {
	Sprite    Sprite
	Transform Transform
	Layer     Layer
	ColorTint ColorTint
	Anchor    Anchor
	Size      ecs.Option[Size]
}

func renderSpritesSystem(screen RenderTarget, sprites ecs.Query[RenderSpritesValue]) {
	items := slices.Collect(sprites.Items())

	slices.SortFunc(items, func(a, b RenderSpritesValue) int {
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
		tr := item.Transform

		// apply rotation
		op.GeoM.Rotate(float64(tr.Rotation))

		if hasCustomSize {
			// apply custom size if available
			scale := size.DivEach(imageSize)
			op.GeoM.Scale(scale.X, scale.Y)
		}

		// apply custom size based on transform
		op.GeoM.Scale(tr.Scale.X, tr.Scale.Y)

		// move to target position
		op.GeoM.Translate(tr.Translation.X, tr.Translation.Y)

		// apply color
		op.ColorScale.ScaleWithColor(item.ColorTint)

		screen.DrawImage(item.Sprite.Image, &op)
	}
}

func ImageSizeOf(image *ebiten.Image) Vec {
	return Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}
