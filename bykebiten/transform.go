package bykebiten

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/internal/arch"
	"log/slog"
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

func TransformFromXY(x, y float64) Transform {
	return Transform{
		Scale:       gm.VecOne,
		Translation: gm.Vec{X: x, Y: y},
	}
}

func (t Transform) WithTranslation(translation gm.Vec) Transform {
	t.Translation = translation
	return t
}

func (t Transform) WithRotation(rotation gm.Rad) Transform {
	t.Rotation = rotation
	return t
}

func (t Transform) WithScale(scale gm.Vec) Transform {
	t.Scale = scale
	return t
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

type simpleItems struct {
	// transforms not within the hierarchy
	_ byke.Without[byke.ChildOf]
	_ byke.Without[byke.Children]
	_ byke.Changed[Transform]

	Transform       Transform
	GlobalTransform *GlobalTransform
}

func syncSimpleTransformSystem(query byke.Query[simpleItems]) {
	for item := range query.Items() {
		item.GlobalTransform.Translation = item.Transform.Translation
		item.GlobalTransform.Scale = item.Transform.Scale
		item.GlobalTransform.Rotation = item.Transform.Rotation
	}
}

type rootItems struct {
	// should not have a parent, so it is a root
	byke.Without[byke.ChildOf]

	// but it should have children
	byke.With[byke.Children]

	//	// should have either a change in children or a changed transform,
	//	// otherwise the immediate subtree did not change (it might have on a deeper level)
	//	_ byke.Or[byke.Changed[byke.Children], byke.Changed[Transform]]

	Children        byke.Option[byke.Children]
	Transform       Transform
	GlobalTransform *GlobalTransform
}

type transformItem struct {
	Children        byke.Option[byke.Children]
	Transform       Transform
	GlobalTransform *GlobalTransform
}

func propagateTransformSystem(
	rootItemsQuery byke.Query[rootItems],
	childItemsQuery byke.Query[transformItem],
) {
	var recurse func(entityId byke.EntityId, parentTransform *GlobalTransform)

	recurse = func(entityId byke.EntityId, parentTransform *GlobalTransform) {
		entity, ok := childItemsQuery.Get(entityId)
		if !ok {
			slog.Warn("Transform hierarchy broken, missing entity", slog.Int("entityId", int(entityId)))
			return
		}

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
