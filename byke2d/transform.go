package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/pulse/glm"
)

type Transform struct {
	byke.ComparableComponent[Transform]
	Translation glm.Vec3f
	Scale       glm.Vec3f
	Rotation    glm.Rad
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

func (Transform) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		GlobalTransform{Scale: glm.Vec3f{1, 1, 1}},
	}
}

type GlobalTransform struct {
	byke.ComparableComponent[GlobalTransform]
	Translation glm.Vec3f
	Scale       glm.Vec3f
	Rotation    glm.Rad
}

func (t GlobalTransform) AsMat3f() glm.Mat3f {
	return glm.RotationMat3[float32](t.Rotation).
		Scale(t.Scale.XY()).
		Translate(t.Translation.Scale(1).XY())
}

func (t GlobalTransform) Mul(other Transform) GlobalTransform {
	affine := t.AsMat3f()
	
	// FIXME clean this up and do it correctly!
	translation := affine.Transform(other.Translation.Truncate().Extend(1.0))

	return GlobalTransform{
		Translation: translation,
		Scale:       t.Scale.Mul(other.Scale),
		Rotation:    t.Rotation + other.Rotation,
	}
}
