package byke2d

import (
	"github.com/oliverbestmann/byke"
)

type MeshInstance struct {
	byke.Component[MeshInstance]
	Mesh *Mesh
}

func (*MeshInstance) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
		InheritVisibility,
	}
}
