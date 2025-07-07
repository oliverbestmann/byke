package main

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/internal/arch"
)

var _ = byke.ValidateComponent[Transform]()
var _ = byke.ValidateComponent[GlobalTransform]()

type Transform struct {
	byke.ComparableComponent[Transform]
	Translation gm.Vec
	Scale       gm.Vec
	Rotation    gm.Rad
}

func NewTransform() Transform {
	return Transform{
		Scale: gm.VecOne,
	}
}

func (t Transform) AsAffine() gm.Affine {
	return gm.Affine{
		Matrix:      gm.ScaleMat(t.Scale).Mul(gm.RotationMat(t.Rotation)),
		Translation: t.Translation,
	}
}

func (Transform) RequireComponents() []arch.ErasedComponent {
	return []arch.ErasedComponent{
		GlobalTransform{},
	}
}

type GlobalTransform struct {
	byke.ComparableComponent[GlobalTransform]
	Translation gm.Vec
	Scale       gm.Vec
	Rotation    gm.Rad
}

func (t GlobalTransform) AsAffine() gm.Affine {
	return gm.Affine{
		Matrix:      gm.ScaleMat(t.Scale).Mul(gm.RotationMat(t.Rotation)),
		Translation: t.Translation,
	}
}

func (t GlobalTransform) Mul(other Transform) GlobalTransform {
	affine := t.AsAffine()
	translation := affine.Transform(other.Translation)

	return GlobalTransform{
		Translation: translation,
		Scale:       t.Scale.MulEach(other.Scale),
		Rotation:    t.Rotation + other.Rotation,
	}
}

type rootItems struct {
	byke.Without[byke.ChildOf]

	Children        byke.Option[byke.Children]
	Transform       Transform
	GlobalTransform *GlobalTransform
}

type childItemsQuery struct {
	Children        byke.Option[byke.Children]
	Transform       Transform
	GlobalTransform *GlobalTransform
}

func propagateTransformSystem(
	rootItemsQuery byke.Query[rootItems],
	childItemsQuery byke.Query[childItemsQuery],
) {
	var recurse func(entityId byke.EntityId, parentTransform *GlobalTransform)

	recurse = func(entityId byke.EntityId, parentTransform *GlobalTransform) {
		entity, _ := childItemsQuery.Get(entityId)

		newTransform := parentTransform.Mul(entity.Transform)
		*entity.GlobalTransform = newTransform

		// recurse into children
		children, ok := entity.Children.Get()
		if ok {
			for _, child := range children.Children() {
				recurse(child, entity.GlobalTransform)
			}
		}
	}

	for root := range rootItemsQuery.Items() {
		// copy directly on root level
		root.GlobalTransform.Translation = root.Transform.Translation
		root.GlobalTransform.Scale = root.Transform.Scale
		root.GlobalTransform.Rotation = root.Transform.Rotation

		// recurse into children
		children, ok := root.Children.Get()
		if ok {
			for _, child := range children.Children() {
				recurse(child, root.GlobalTransform)
			}
		}
	}
}
