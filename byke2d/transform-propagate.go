package byke2d

import (
	"log/slog"

	"github.com/oliverbestmann/byke"
)

type simpleTransformItems struct {
	// transforms not within the hierarchy
	_ byke.Without[byke.ChildOf]
	_ byke.Without[byke.Children]
	_ byke.Changed[Transform]

	Transform       Transform
	GlobalTransform *GlobalTransform
}

type rootTransformItems struct {
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

func syncSimpleTransformSystem(query byke.Query[simpleTransformItems]) {
	for item := range query.Items() {
		item.GlobalTransform.Translation = item.Transform.Translation
		item.GlobalTransform.Scale = item.Transform.Scale
		item.GlobalTransform.Rotation = item.Transform.Rotation
	}
}

func propagateTransformSystem(
	rootItemsQuery byke.Query[rootTransformItems],
	childItemsQuery byke.Query[transformItem],
) {
	var recurse func(entityId byke.EntityId, parentTransform *GlobalTransform)

	recurse = func(entityId byke.EntityId, parentTransform *GlobalTransform) {
		entity, ok := childItemsQuery.Get(entityId)
		if !ok {
			slog.Warn(
				"Transform hierarchy broken, missing entity",
				slog.Int("entityId", int(entityId)),
			)

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
