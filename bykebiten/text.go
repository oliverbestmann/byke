package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	. "github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/assets"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/internal/arch"
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
		TextFace{Face: DefaultFontFace()},
		Layer{},
		Transform{},
		ColorTint{Color: color.White},
		AnchorCenter,
		ComputedSize{},
	}
}

type TextFace struct {
	ComparableComponent[TextFace]
	text.Face
}
