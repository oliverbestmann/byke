package byke2d

import (
	"fmt"
	"strings"

	"github.com/oliverbestmann/byke"
)

func dumpTree(
	rootsQuery byke.Query[struct {
		_        byke.Without[byke.ChildOf]
		EntityId byke.EntityId
	}],
	nodesQuery byke.Query[struct {
		EntityRef byke.EntityRef
		Name      byke.Option[byke.Name]
		Children  byke.Option[byke.Children]
	}]) {

	// recursive declaration
	var dumpTree func(level int, nodeId byke.EntityId)

	dumpTree = func(level int, nodeId byke.EntityId) {
		var line strings.Builder

		for range level {
			line.WriteString("  ")
		}

		node, ok := nodesQuery.Get(nodeId)
		if !ok {
			line.WriteString("<missing>")
			fmt.Println(line.String())
			return
		}

		if name, ok := node.Name.Get(); ok {
			line.WriteString(name.String())
		} else {
			line.WriteString("<node>")
		}

		line.WriteString(" (")
		for _, comp := range node.EntityRef.Components() {
			line.WriteString(comp.ComponentType().Type.Name())
			line.WriteString(",")
		}
		line.WriteString(")")

		fmt.Println(line.String())

		if children, ok := node.Children.Get(); ok {
			for _, child := range children.Children() {
				dumpTree(level+1, child)
			}
		}

	}
	for root := range rootsQuery.Items() {
		fmt.Println()
		dumpTree(0, root.EntityId)
	}
}
