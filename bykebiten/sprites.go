package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
)

var _ = byke.ValidateComponent[Sprite]()

type Sprite struct {
	byke.ComparableComponent[Sprite]
	Image      *ebiten.Image
	CustomSize *gm.Vec
}

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return commonRenderComponents
}

func computeSpriteSizeSystem(
	query byke.Query[struct {
		byke.Changed[Sprite]

		BBox   *BBox
		Sprite Sprite
		Anchor Anchor
	}],
) {
	for item := range query.Items() {
		size := imageSizeOf(item.Sprite.Image)
		origin := item.Anchor.MulEach(size)
		item.BBox.Rect = gm.RectWithOriginAndSize(origin, size)
	}
}
