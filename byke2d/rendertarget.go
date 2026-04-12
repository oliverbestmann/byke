package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
)

var _ = byke.ValidateComponent[RenderTarget]()

type RenderTarget struct {
	byke.Component[RenderTarget]

	// Render to the primary window
	PrimaryWindow bool

	// Render to a specific texture
	Texture *Texture
}

func (RenderTarget) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{ViewTarget{}}
}

var PrimaryWindowRenderTarget = RenderTarget{PrimaryWindow: true}
