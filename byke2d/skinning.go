package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

var _ = byke.ValidateComponent[SkinnedMesh]()

type SkinnedMesh struct {
	byke.Component[SkinnedMesh]
	InverseBind []glm.Mat4f
	Joints      []byke.EntityId
}
