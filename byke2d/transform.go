package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
)

type Transform struct {
	byke.Component[Transform]
	Translation glm.Vec3f
}

func NewTransform() Transform {
	return Transform{}
}

type GlobalTransform struct {
	byke.Component[GlobalTransform]
	Translation glm.Vec3f
}

func (t GlobalTransform) AsAffine() glm.Mat4f {
	// TODO
	return glm.IdentityMat4[float32]()
}
