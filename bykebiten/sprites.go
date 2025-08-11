package bykebiten

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Tiles]()
var _ = byke.ValidateComponent[TileIndex]()
var _ = byke.ValidateComponent[tileCache]()

type Sprite struct {
	byke.ComparableComponent[Sprite]
	Image *ebiten.Image

	CustomSize Optional[gm.Vec]

	// flips the sprite during rendering.
	FlipX, FlipY bool
}

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return append(
		[]byke.ErasedComponent{AnchorCenter, Blend{}, Filter{}},
		commonRenderComponents...,
	)
}

type TileIndex struct {
	byke.ComparableComponent[TileIndex]
	Index int
}

type Tiles struct {
	byke.ComparableComponent[Tiles]
	Columns uint16
	Rows    uint16
	Width   uint16
	Height  uint16
	OffsetX uint16
	OffsetY uint16
	GapX    uint16
	GapY    uint16
}

func (*Tiles) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		TileIndex{},
		tileCache{},
	}
}

func (t *Tiles) IsValid() bool {
	return t.Width > 0 && t.Height > 0
}

func (t *Tiles) Frame(n int) image.Rectangle {
	// clamp n into the valid range
	n = min(max(n, 0), int(t.Rows*t.Columns-1))

	row := n / int(t.Columns)
	column := n % int(t.Columns)

	x0 := column*int(t.Width+t.GapX) + int(t.OffsetX)
	y0 := row*int(t.Height+t.GapY) + int(t.OffsetY)
	x1 := x0 + int(t.Width)
	y1 := y0 + int(t.Height)

	return image.Rect(x0, y0, x1, y1)
}

func (t *Tiles) Count() int {
	return int(t.Rows * t.Columns)
}

type tileCache struct {
	byke.Component[tileCache]
	Tiles []*ebiten.Image
}

func updateTileCache(
	query byke.Query[struct {
		_ byke.Or[byke.Changed[Tiles], byke.Changed[Sprite]]

		Tiles  Tiles
		Sprite Sprite
		Cache  *tileCache
	}],
) {
	for item := range query.Items() {
		// reuse tiles array
		clear(item.Cache.Tiles)
		item.Cache.Tiles = item.Cache.Tiles[:0]

		for idx := range item.Tiles.Count() {
			subImage := item.Sprite.Image.SubImage(item.Tiles.Frame(idx)).(*ebiten.Image)
			item.Cache.Tiles = append(item.Cache.Tiles, subImage)
		}
	}
}

func computeSpriteSizeSystem(
	query byke.Query[struct {
		_ byke.OrStruct[struct {
			_ byke.Changed[Tiles]
			_ byke.Changed[Sprite]
			_ byke.Changed[Anchor]
		}]

		BBox   *BBox
		Sprite Sprite
		Anchor Anchor

		Tiles byke.Option[Tiles]
	}],
) {
	for item := range query.Items() {
		var sourceSize, targetSize gm.Vec

		if tiles, ok := item.Tiles.Get(); ok {
			sourceSize = gm.Vec{X: float64(tiles.Width), Y: float64(tiles.Height)}
		} else if item.Sprite.Image != nil {
			sourceSize = imageSizeOf(item.Sprite.Image)
		} else {
			sourceSize = gm.VecOne
		}

		if item.Sprite.CustomSize.IsSet {
			targetSize = item.Sprite.CustomSize.Value
		} else {
			targetSize = sourceSize
		}

		if item.Sprite.FlipX {
			sourceSize.X *= -1
		}

		if item.Sprite.FlipY {
			sourceSize.Y *= -1
		}

		// the bounding box is in "custom size" scale. this is not the same as the
		// image size.
		origin := item.Anchor.MulEach(targetSize).Mul(-1)
		item.BBox.Rect = gm.RectWithOriginAndSize(origin, targetSize)

		// record a factor to go from bbox to image scale.
		item.BBox.ToSourceScale = sourceSize.DivEach(targetSize)
	}
}
