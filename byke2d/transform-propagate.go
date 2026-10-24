package byke2d

import (
	"cmp"
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
	_ byke.Without[byke.ChildOf]

	// and have children (otherwise it would be caught by simpleTransform)
	Children byke.Children

	Transform       Transform
	GlobalTransform *GlobalTransform

	EntityId byke.EntityId
}

type transformItem struct {
	Children        byke.Option[byke.Children]
	Transform       Transform
	GlobalTransform *GlobalTransform
}

func syncSimpleTransformSystem(query byke.Query[simpleTransformItems]) {
	for item := range query.Items() {
		item.GlobalTransform.Affine = item.Transform.Affine3()
	}
}

func propagateTransformSystem(
	rootItemsQuery byke.Query[rootTransformItems],
	childItemsQuery byke.Query[transformItem],
	nameQuery byke.Query[byke.Name],
) {
	var recurse func(parentId, entityId byke.EntityId, parentTransform *GlobalTransform)

	recurse = func(parentId, entityId byke.EntityId, parentTransform *GlobalTransform) {
		entity, ok := childItemsQuery.Get(entityId)
		if !ok {
			// try to get the name of the parent
			parent, _ := nameQuery.Get(parentId)

			slog.Warn(
				"Transform hierarchy broken, missing entity",
				slog.Int("entityId", int(entityId)),
				slog.Int("parentId", int(parentId)),
				slog.String("parent", cmp.Or(parent.Name, "Unknown")),
			)

			return
		}

		newTransform := parentTransform.Mul(entity.Transform)
		*entity.GlobalTransform = newTransform

		// recurse into children
		children, ok := entity.Children.Get()
		if ok {
			for _, child := range children.Children() {
				recurse(entityId, child, entity.GlobalTransform)
			}
		}
	}

	for root := range rootItemsQuery.Items() {
		// copy directly on root level
		root.GlobalTransform.Affine = root.Transform.Affine3()

		// recurse into children
		for _, child := range root.Children.Children() {
			recurse(root.EntityId, child, root.GlobalTransform)
		}
	}
}
