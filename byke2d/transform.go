package byke2d

import (
	"log/slog"

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
		Translate(t.Translation.Scale(-1).XY())
}

func (t GlobalTransform) Mul(other Transform) GlobalTransform {
	affine := t.AsMat3f()
	translation := affine.Transform(other.Translation)

	return GlobalTransform{
		Translation: translation,
		Scale:       t.Scale.Mul(other.Scale),
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

type rootItems struct {
	// should not have a parent, so it is a root
	byke.Without[byke.ChildOf]

	// but it should have children
	byke.With[byke.Children]

	Children        byke.Option[byke.Children]
	Transform       Transform
	GlobalTransform *GlobalTransform
}

type transformItem struct {
	Children        byke.Option[byke.Children]
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
