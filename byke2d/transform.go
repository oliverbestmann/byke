package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/spoke"
)

var (
	_ = byke.ValidateComponent[Transform]()
	_ = byke.ValidateComponent[GlobalTransform]()
)

type Transform struct {
	byke.ComparableComponent[Transform]
	Translation glm.Vec3f
	Scale       glm.Vec3f
	Rotation    glm.Quat
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
	t.Scale = glm.Vec3f{x, y, 1}
	return t
}

func (t Transform) WithScaleXYZ(x, y, z float32) Transform {
	t.Scale = glm.Vec3f{x, y, z}
	return t
}

func (t Transform) WithRotation(rotation glm.Quat) Transform {
	t.Rotation = rotation
	return t
}

func (t Transform) WithRotationX(rotation glm.Rad) Transform {
	t.Rotation = glm.RotationXQuat(rotation)
	return t
}

func (t Transform) WithRotationY(rotation glm.Rad) Transform {
	t.Rotation = glm.RotationYQuat(rotation)
	return t
}

func (t Transform) WithRotationZ(rotation glm.Rad) Transform {
	t.Rotation = glm.RotationZQuat(rotation)
	return t
}

func (t Transform) Affine3() glm.Mat4f {
	mat := glm.TranslationMat4f(t.Translation.XYZ())
	mat.RotateAssign(t.Rotation)
	mat.ScaleAssign(t.Scale.XYZ())
	return mat
}

func (Transform) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		GlobalTransform{},
	}
}

type GlobalTransform struct {
	byke.ComparableComponent[GlobalTransform]

	// TODO we should make this an actual "affine" type at some point
	//  to ensure we can only do affine transformations here
	Affine glm.Mat4f
}

func (t GlobalTransform) Mul(other Transform) GlobalTransform {
	mat := t.Affine
	mat.TranslateAssign(other.Translation.XYZ())
	mat.RotateAssign(other.Rotation)
	mat.ScaleAssign(other.Scale.XYZ())
	return GlobalTransform{Affine: mat}
}
