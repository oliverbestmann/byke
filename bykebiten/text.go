package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	. "github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/assets"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/internal/arch"
	"slices"
	"sync"
)

var DefaultFontFace = sync.OnceValue(func() text.Face {
	return &text.GoTextFace{
		Source: assets.FiraMono(),
		Size:   16.0,
	}
})

type Text struct {
	ComparableComponent[Text]
	Text string
}

func (t Text) RequireComponents() []arch.ErasedComponent {
	return []ErasedComponent{
		Face{Face: DefaultFontFace()},
		Layer{},
		Transform{},
		ColorTint{Color: color.White},
		AnchorCenter,
	}
}

type Face struct {
	ComparableComponent[Face]
	text.Face
}

type renderTextQueryItem struct {
	Text      Text
	Face      Face
	Transform GlobalTransform
	Anchor    Anchor
	Layer     Layer
	ColorTint ColorTint
}

func renderTextSystem(
	itemsCache *Local[[]renderTextQueryItem],
	renderTarget RenderTarget,
	query Query[renderTextQueryItem],
) {
	itemsCache.Value = itemsCache.Value[:0]

	for item := range query.Items() {
		itemsCache.Value = append(itemsCache.Value, item)
	}

	// sort by z component
	slices.SortFunc(itemsCache.Value, func(a, b renderTextQueryItem) int {
		switch {
		case a.Layer.Z < b.Layer.Z:
			return -1

		case a.Layer.Z > b.Layer.Z:
			return 1

		default:
			return 0
		}
	})

	for _, item := range itemsCache.Value {
		lineSpacing := item.Face.Metrics().VLineGap
		textSize := gm.VecOf(text.Measure(item.Text.Text, item.Face.Face, lineSpacing))

		pos := item.Transform.Translation.Sub(textSize.MulEach(item.Anchor.Vec))

		ops := text.DrawOptions{}
		ops.LineSpacing = lineSpacing

		// TODO use GeoM similar to drawSprite
		ops.GeoM.Translate(pos.X, pos.Y)

		text.Draw(renderTarget.Image, item.Text.Text, item.Face, &ops)
	}
}
