package byke2d

import (
	"github.com/oliverbestmann/byke"
)

var _ = byke.ValidateComponent[RenderTarget]()

type RenderTarget struct {
	byke.Component[RenderTarget]

	// Render to the primary window
	PrimaryWindow bool

	// Render to a specific texture
	Texture *RenderTexture
}

var PrimaryWindowRenderTarget = RenderTarget{PrimaryWindow: true}
