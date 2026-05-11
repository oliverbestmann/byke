package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
)

var _ = byke.ValidateComponent[HDR]()

type HDR struct {
	byke.ImmutableComponent[HDR]
}

func (HDR) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		DefaultColorGrading,
		TonemappingTonyMcMapface,
		DebandDitherOn,
		BloomNatural,
	}
}
