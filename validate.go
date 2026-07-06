package byke

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	pendingValidationsLock    sync.Mutex
	pendingValidationsCleared bool
	pendingValidations        []func()
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
// This identifies mistakes in the type passed to Component during compile time,
// and some more mistake directly at startup.
func ValidateComponent[C IsComponent[C]]() struct{} {
	pendingValidationsLock.Lock()
	defer pendingValidationsLock.Unlock()

	validate := func() {
		var cZero C
		validateComponent(cZero)
	}

	if pendingValidationsCleared {
		// new wold was already called, validating directly
		validate()
		return struct{}{}
	}

	pendingValidations = append(pendingValidations, validate)

	return struct{}{}
}

func flushComponentValidations() {
	pendingValidationsLock.Lock()
	defer pendingValidationsLock.Unlock()

	if pendingValidationsCleared {
		return
	}

	for _, validate := range pendingValidations {
		validate()
	}

	pendingValidationsCleared = true
}

func validateComponent(c ErasedComponent) {
	componentType := c.ComponentType()

	if parent, ok := componentType.New().(isRelationshipTargetType); ok {
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

	if child, ok := componentType.New().(isRelationshipComponent); ok {
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
}
