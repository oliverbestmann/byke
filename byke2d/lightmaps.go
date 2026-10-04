package byke2d

import "github.com/oliverbestmann/byke"

var _ = byke.ValidateComponent[Lightmap]()

type Lightmap struct {
	byke.Component[Lightmap]

	Image   *Texture
	Bicubic bool
}
