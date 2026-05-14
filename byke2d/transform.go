package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/pulse/glm"
)

var _ = byke.ValidateComponent[Transform]()
var _ = byke.ValidateComponent[GlobalTransform]()

type Transform struct {
	byke.ComparableComponent[Transform]
	Translation glm.Vec3f
	Scale       glm.Vec3f

	// TODO we need a something like glm.Quad4 at some point
	Rotation glm.Rad
}

func NewTransform() Transform {
	return Transform{
		Scale: glm.Vec3f{1, 1, 1},
	}
}

func TransformFromXY(x, y float32) Transform {
	return TransformFromXYZ(x, y, 0)
}

func TransformFromXYZ(x, y, z float32) Transform {
	return Transform{
		Scale:       glm.Vec3f{1, 1, 1},
		Translation: glm.Vec3f{x, y, z},
	}
}

func (t Transform) WithTranslationXY(x, y float32) Transform {
	t.Translation = glm.Vec3f{x, y, 0}
	return t
}
func (t Transform) WithTranslationXYZ(x, y, z float32) Transform {
	t.Translation = glm.Vec3f{x, y, z}
	return t
}

func (t Transform) WithScaleXY(x, y float32) Transform {
	t.Scale = glm.Vec3f{x, y, 0}
	return t
}

func (t Transform) WithScaleXYZ(x, y, z float32) Transform {
	t.Scale = glm.Vec3f{x, y, z}
	return t
}

func (t Transform) WithRotation(rotation glm.Rad) Transform {
	t.Rotation = rotation
	return t
}

func (t Transform) Affine2() glm.Mat3f {
	return glm.TranslationMat3[float32](t.Translation.XY()).
		Rotate(t.Rotation).
		Scale(t.Scale.XY())

	// return glm.RotationMat3[float32](t.Rotation).
	// 	Scale(t.Scale.XY()).
	// 	Translate(t.Translation.Scale(1).XY())
}

func (Transform) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		GlobalTransform{},
	}
}

type GlobalTransform struct {
	byke.ComparableComponent[GlobalTransform]
	Affine glm.Mat3f
}

func (t GlobalTransform) Mul(other Transform) GlobalTransform {
	return GlobalTransform{
		Affine: t.Affine.Mul(other.Affine2()),
	}
}
