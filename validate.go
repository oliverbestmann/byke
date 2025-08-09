package byke

import (
	"fmt"
	"reflect"

	"github.com/oliverbestmann/byke/spoke"
)

// ValidateComponent should be called to verify that the IsComponent interface is correctly implemented.
//
//	type Position struct {
//	   Component[Position]
//	   X, Y float64
//	}
//
//	var _ = ValidateComponent[Position]()
//
// This identifies mistakes in the type passed to Component during compile time.
func ValidateComponent[C IsComponent[C]]() struct{} {
	componentType := spoke.ComponentTypeOf[C]()

	var zero C

	if parent, ok := any(zero).(isRelationshipTargetType); ok {
		// check if the child type points to us
		childType := parent.RelationshipType()
		instance := reflect.New(childType.Type).Elem().Interface()

		child, ok := instance.(isRelationshipComponent)
		if !ok {
			panic(fmt.Sprintf(
				"relationship target of %s must point to a component embedding byke.Relationship",
				componentType,
			))
		}

		if child.RelationshipTargetType() != componentType {
			panic(fmt.Sprintf(
				"relationship target of %s must point to %s",
				childType, componentType,
			))
		}
	}

	if child, ok := any(zero).(isRelationshipComponent); ok {
		// check if the child type points to us
		parentType := child.RelationshipTargetType()

		parentComponent := parentType.New()
		parent, ok := parentComponent.(isRelationshipTargetType)
		if !ok {
			panic(fmt.Sprintf(
				"relationship target of %s must point to a component embedding byke.RelationshipTarget",
				componentType,
			))
		}

		if parent.RelationshipType() != componentType {
			panic(fmt.Sprintf(
				"relationship target of %s must point to %s",
				parentType, componentType,
			))
		}
	}

	// TODO mark component as valid somewhere, maybe calculate some
	//  kind of component type id too
	return struct{}{}
}
