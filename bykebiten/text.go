package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	. "github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/assets"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
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

var textRequiredComponents = sync.OnceValue(func() []spoke.ErasedComponent {
	components := []ErasedComponent{
		&TextFace{Face: DefaultFontFace()},
		AnchorCenter,
	}

	return append(components, commonRenderComponents...)
})

func (Text) RequireComponents() []spoke.ErasedComponent {
	return textRequiredComponents()
}

type TextFace struct {
	ComparableComponent[TextFace]
	text.Face
}

func computeTextSizeSystem(
	query Query[struct {
		Or[Changed[Text], Changed[TextFace]]

		BBox     *BBox
		Text     Text
		TextFace TextFace
		Anchor   Anchor
	}],
) {
	for item := range query.Items() {
		lineSpacing := item.TextFace.Face.Metrics().VLineGap
		size := gm.VecOf(text.Measure(item.Text.Text, item.TextFace.Face, lineSpacing))

		origin := item.Anchor.MulEach(size).Mul(-1)
		item.BBox.Rect = gm.RectWithOriginAndSize(origin, size)
		item.BBox.ToSourceScale = gm.VecOne
	}
}
