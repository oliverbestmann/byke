package byke2d

import (
	"github.com/oliverbestmann/byke"
)

func syncSimpleVisibilitySystem(
	query byke.Query[struct {
		_ byke.Without[byke.ChildOf]
		_ byke.Without[byke.Children]
		_ byke.Changed[Visibility]

		Visibility         Visibility
		ComputedVisibility *ComputedVisibility
	}],
) {
	for item := range query.Items() {
		*item.ComputedVisibility = item.Visibility.Compute(
			ComputedVisibility{Visible: true},
		)
	}
}

func propagateVisibilitySystem(
	world *byke.World,
	roots byke.Query[struct {
		_ byke.Without[byke.ChildOf]
		_ byke.With[byke.Children]

		Visibility         Visibility
		ComputedVisibility *ComputedVisibility
		Children           byke.Children
	}],

	nodes byke.Query[struct {
		Visibility         Visibility
		ComputedVisibility *ComputedVisibility
		Children           byke.Option[byke.Children]
	}],
) {
	var propagateVisibility func(nodeId byke.EntityId, parentVisibility ComputedVisibility)

	propagateVisibility = func(nodeId byke.EntityId, parentVisibility ComputedVisibility) {
		node, ok := nodes.Get(nodeId)
		if !ok {
			// Node does not exist or has no Visibility or ComputedVisibility field.
			// A common case where this happens is a Light that is attached to a Mesh entity.
			return
		}

		*node.ComputedVisibility = node.Visibility.Compute(parentVisibility)

		children, ok := node.Children.Get()
		if !ok {
			return
		}

		for _, child := range children.Children() {
			propagateVisibility(child, *node.ComputedVisibility)
		}
	}

	for root := range roots.Items() {
		*root.ComputedVisibility = root.Visibility.Compute(
			ComputedVisibility{Visible: true},
		)

		for _, child := range root.Children.Children() {
			propagateVisibility(child, *root.ComputedVisibility)
		}
	}
}
