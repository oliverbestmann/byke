package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
	"image"
	"slices"
	"sync"
)

var whiteImage = sync.OnceValue(func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
})

var _ = byke.ValidateComponent[Layer]()
var _ = byke.ValidateComponent[ColorTint]()
var _ = byke.ValidateComponent[Anchor]()
var _ = byke.ValidateComponent[BBox]()

type Layer struct {
	byke.ComparableComponent[Layer]
	Z float64
}

type Anchor struct {
	byke.ComparableComponent[Anchor]
	gm.Vec
}

var (
	AnchorTopLeft      = Anchor{Vec: gm.Vec{}}
	AnchorTopCenter    = Anchor{Vec: gm.Vec{X: 0.5}}
	AnchorTopRight     = Anchor{Vec: gm.Vec{X: 1.0}}
	AnchorCenterLeft   = Anchor{Vec: gm.Vec{X: 0, Y: 0.5}}
	AnchorCenter       = Anchor{Vec: gm.Vec{X: 0.5, Y: 0.5}}
	AnchorCenterRight  = Anchor{Vec: gm.Vec{X: 1.0, Y: 0.5}}
	AnchorBottomLeft   = Anchor{Vec: gm.Vec{Y: 1.0}}
	AnchorBottomCenter = Anchor{Vec: gm.Vec{X: 0.5, Y: 1.0}}
	AnchorBottomRight  = Anchor{Vec: gm.Vec{X: 1.0, Y: 1.0}}
)

type ColorTint struct {
	byke.ComparableComponent[ColorTint]
	color.Color
}

type RenderTarget struct {
	*ebiten.Image
}

type BBox struct {
	byke.ComparableComponent[BBox]
	gm.Rect

	// the bounding box might not reflect the actual source, e.g. for sprites
	// that have a custom size set. ToSourceScale describes the factor that
	// the Rect would need to be multiplied with to get the sources size.
	ToSourceScale gm.Vec
	LocalOrigin   gm.Vec
}

var commonRenderComponents = []byke.ErasedComponent{
	NewTransform(),
	Layer{},
	AnchorCenter,
	ColorTint{Color: color.White},
	BBox{},
}

type renderCommonValues struct {
	BBox      BBox
	ColorTint ColorTint
	Layer     Layer
	Transform GlobalTransform
}

type hasCommonValues interface {
	commonValues() *renderCommonValues
}

type renderSpriteValue struct {
	Common renderCommonValues
	Sprite Sprite

	TileIndex byke.Option[TileIndex]
	TileCache byke.Option[tileCache]
}

func (r *renderSpriteValue) commonValues() *renderCommonValues {
	return &r.Common
}

type renderTextValue struct {
	Common renderCommonValues
	Text   Text
	Face   TextFace
}

func (r *renderTextValue) commonValues() *renderCommonValues {
	return &r.Common
}

type renderVectorValue struct {
	Common renderCommonValues
	Path   Path
	Fill   byke.Option[Fill]
	Stroke byke.Option[Stroke]
}

func (r *renderVectorValue) commonValues() *renderCommonValues {
	return &r.Common
}

type renderCache struct {
	Sprites []renderSpriteValue
	Texts   []renderTextValue
	Vectors []renderVectorValue

	TempPath vector.Path

	// scratch space for vertex transformations
	vertexCache []ebiten.Vertex

	all []hasCommonValues
}

func renderSystem(
	screen RenderTarget,
	spritesQuery byke.Query[renderSpriteValue],
	textsQuery byke.Query[renderTextValue],
	vectorsQuery byke.Query[renderVectorValue],
	cache *byke.Local[renderCache],
) {
	c := &cache.Value

	defer func() {
		clear(c.Sprites)
		clear(c.Texts)
		clear(c.Vectors)
		clear(c.all)
	}()

	// re-use the slices and collect all values
	c.Sprites = slices.AppendSeq(c.Sprites[:0], spritesQuery.Items())
	c.Texts = slices.AppendSeq(c.Texts[:0], textsQuery.Items())
	c.Vectors = slices.AppendSeq(c.Vectors[:0], vectorsQuery.Items())

	items := c.Items()

	// sort sprites by layer
	slices.SortFunc(items, func(a, b hasCommonValues) int {
		return compareZ(a.commonValues(), b.commonValues())
	})

	for _, item := range items {
		common := item.commonValues()

		var g ebiten.GeoM

		// get transformation
		tr := common.Transform

		// custom scale, e.g. derived from sprites CustomSize property
		toSourceScale := common.BBox.ToSourceScale
		if toSourceScale != gm.VecOne {
			g.Scale(1/toSourceScale.X, 1/toSourceScale.Y)
		}

		// transform by offset. Need to multiply by the sign of source scale as that might
		// flip the direction the origin translation need to be applied
		origin := common.BBox.Min
		g.Translate(origin.X*signOf(toSourceScale.X), origin.Y*signOf(toSourceScale.Y))

		// apply flip values
		// g.Scale(common.BBox.FlipScale.X, toSourceScale.Y)

		if tr.Scale != gm.VecOne {
			// apply custom size based on transform
			g.Scale(tr.Scale.X, tr.Scale.Y)
		}

		if tr.Rotation != 0 {
			// apply rotation
			g.Rotate(float64(tr.Rotation))
		}

		// move to target position
		g.Translate(tr.Translation.X, tr.Translation.Y)

		// apply color
		var colorScale ebiten.ColorScale
		colorScale.Scale(common.ColorTint.PremultipliedValues())

		switch item := item.(type) {
		case *renderSpriteValue:
			image := item.Sprite.Image
			if tileCache_, ok := item.TileCache.Get(); ok {
				var tileCache tileCache = tileCache_

				idx := item.TileIndex.OrZero().Index
				image = tileCache.Tiles[idx%len(tileCache.Tiles)]
			}

			var op ebiten.DrawImageOptions
			op.GeoM = g
			op.ColorScale = colorScale
			screen.DrawImage(image, &op)

		case *renderTextValue:
			var op text.DrawOptions
			op.GeoM = g
			op.ColorScale = colorScale
			op.LineSpacing = item.Face.Metrics().VLineGap
			text.Draw(screen.Image, item.Text.Text, item.Face, &op)

		case *renderVectorValue:
			// need to pre translate using the paths origin to make it match the
			// actual render origin
			var pathG ebiten.GeoM
			pathG.Translate(-origin.X, -origin.Y)
			pathG.Concat(g)

			path := &cache.Value.TempPath
			path.Reset()
			path.AddPath(item.Path.inner(), &vector.AddPathOptions{GeoM: pathG})

			if fill_, ok := item.Fill.Get(); ok {
				var fill Fill = fill_
				vector.FillPath(screen.Image, path, fill.Color, fill.Antialias, vector.FillRuleNonZero)
			}

			if stroke_, ok := item.Stroke.Get(); ok {
				var stroke Stroke = stroke_

				vector.StrokePath(screen.Image, path, stroke.Color, stroke.Antialias, &vector.StrokeOptions{
					Width:      float32(stroke.Width),
					LineCap:    stroke.LineCap,
					LineJoin:   stroke.LineJoin,
					MiterLimit: float32(stroke.MiterLimit),
				})
			}
		}
	}
}

func (c *renderCache) Items() []hasCommonValues {
	c.all = c.all[:0]

	for idx := range c.Sprites {
		c.all = append(c.all, &c.Sprites[idx])
	}

	for idx := range c.Texts {
		c.all = append(c.all, &c.Texts[idx])
	}

	for idx := range c.Vectors {
		c.all = append(c.all, &c.Vectors[idx])
	}

	return c.all
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

func signOf(x float64) float64 {
	if x < 0 {
		return -1
	} else {
		return 1
	}
}
