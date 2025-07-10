package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
	"slices"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Layer]()
var _ = byke.ValidateComponent[ComputedSize]()
var _ = byke.ValidateComponent[ColorTint]()
var _ = byke.ValidateComponent[Anchor]()

type Sprite struct {
	byke.ComparableComponent[Sprite]
	Image      *ebiten.Image
	CustomSize *gm.Vec
}

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		Layer{},
		Transform{
			Scale: gm.VecOf(1.0, 1.0),
		},
		AnchorCenter,
		ComputedSize{},
		ColorTint{Color: color.White},
	}
}

type ComputedSize struct {
	byke.ComparableComponent[ComputedSize]
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

var AnchorTopLeft = Anchor{Vec: gm.Vec{}}
var AnchorCenter = Anchor{Vec: gm.Vec{X: 0.5, Y: 0.5}}

type ColorTint struct {
	byke.ComparableComponent[ColorTint]
	color.Color
}

type RenderTarget struct {
	*ebiten.Image
}

func ImageSizeOf(image *ebiten.Image) gm.Vec {
	return gm.Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}

type renderCommonValues struct {
	ComputedSize ComputedSize
	Anchor       Anchor
	ColorTint    ColorTint
	Layer        Layer
	Transform    GlobalTransform
}

type renderSpritesValue struct {
	Common renderCommonValues
	Sprite Sprite
}

func (r *renderSpritesValue) commonValues() *renderCommonValues {
	return &r.Common
}

func computeSpriteSizeSystem(
	query byke.Query[struct {
		byke.Changed[Sprite]

		ComputedSize *ComputedSize
		Sprite       Sprite
	}],
) {
	for item := range query.Items() {
		item.ComputedSize.Vec = ImageSizeOf(item.Sprite.Image)
	}
}

func computeTextSizeSystem(
	query byke.Query[struct {
		byke.Or[byke.Changed[Text], byke.Changed[TextFace]]

		ComputedSize *ComputedSize
		Text         Text
		TextFace     TextFace
	}],
) {
	for item := range query.Items() {
		lineSpacing := item.TextFace.Face.Metrics().VLineGap
		size := gm.VecOf(text.Measure(item.Text.Text, item.TextFace.Face, lineSpacing))
		item.ComputedSize.Vec = size
	}
}

type renderTextsValue struct {
	Common renderCommonValues
	Text   Text
	Face   TextFace
}

func (r *renderTextsValue) commonValues() *renderCommonValues {
	return &r.Common
}

type renderCache struct {
	Sprites []renderSpritesValue
	Texts   []renderTextsValue
	Joined  []hasCommonValues
}

func renderSystem(
	screen RenderTarget,
	spritesQuery byke.Query[renderSpritesValue],
	textsQuery byke.Query[renderTextsValue],
	cache *byke.Local[renderCache],
) {
	c := &cache.Value

	defer func() {
		clear(c.Sprites)
		clear(c.Texts)
		clear(c.Joined)
	}()

	// re-use the slice
	c.Sprites = slices.AppendSeq(c.Sprites[:0], spritesQuery.Items())
	c.Texts = slices.AppendSeq(c.Texts[:0], textsQuery.Items())

	c.Joined = c.Joined[:0]

	for idx := range c.Sprites {
		c.Joined = append(c.Joined, &c.Sprites[idx])
	}

	for idx := range c.Texts {
		c.Joined = append(c.Joined, &c.Texts[idx])
	}

	// sort sprites by layer
	slices.SortFunc(c.Joined, func(a, b hasCommonValues) int {
		return compareZ(a.commonValues(), b.commonValues())
	})

	for _, item := range c.Joined {
		common := item.commonValues()

		itemSize := common.ComputedSize.Vec

		var g ebiten.GeoM

		// offset by anchor
		offset := itemSize.MulEach(common.Anchor.Vec)
		g.Translate(-offset.X, -offset.Y)

		// get transformation
		tr := common.Transform

		// apply custom size if available
		scale := itemSize.DivEach(itemSize)
		g.Scale(scale.X, scale.Y)

		// apply custom size based on transform
		g.Scale(tr.Scale.X, tr.Scale.Y)

		// apply rotation
		g.Rotate(float64(tr.Rotation))

		// move to target position
		g.Translate(tr.Translation.X, tr.Translation.Y)

		// apply color
		var colorScale ebiten.ColorScale
		colorScale.Scale(common.ColorTint.PremultipliedValues())

		switch item := item.(type) {
		case *renderSpritesValue:
			var op ebiten.DrawImageOptions
			op.GeoM = g
			op.ColorScale = colorScale
			screen.DrawImage(item.Sprite.Image, &op)

		case *renderTextsValue:
			var op text.DrawOptions
			op.GeoM = g
			op.ColorScale = colorScale
			op.LineSpacing = item.Face.Metrics().VLineGap
			text.Draw(screen.Image, item.Text.Text, item.Face, &op)
		}
	}
}

type hasCommonValues interface {
	commonValues() *renderCommonValues
}

func compareZ(a, b *renderCommonValues) int {
	switch {
	case a.Layer.Z < b.Layer.Z:
		return -1
	case a.Layer.Z > b.Layer.Z:
		return 1
	default:
		return 0
	}
}
