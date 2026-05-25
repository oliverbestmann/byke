package byke2d

import (
	"github.com/oliverbestmann/byke"
)

type Mesh3d struct {
	byke.Component[Mesh3d]
	Mesh *Mesh
}

func (*Mesh3d) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
		InheritVisibility,
	}
}
