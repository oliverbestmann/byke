package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/inner"
	"reflect"
)

type changedAccessor interface {
	inner.HasType
	isChangedAccessor(changedAccessor)
	mutable() bool
	innerValue() reflect.Value
}

type Changed[T IsComparableComponent[T]] struct {
	inner.Type[T]
	Value T
}

type dummy struct{ ComparableComponent[dummy] }

var _ changedAccessor = &Changed[dummy]{}

func (o *Changed[T]) innerValue() reflect.Value {
	return reflect.ValueOf(&o.Value).Elem()
}

func (o *Changed[T]) mutable() bool {
	return false
}

func (o *Changed[T]) isChangedAccessor(changedAccessor) {}

func isChangedType(tyTarget reflect.Type) bool {
	tyChangedAccessor := reflect.TypeFor[changedAccessor]()

	return tyTarget.Kind() != reflect.Pointer &&
		reflect.PointerTo(tyTarget).Implements(tyChangedAccessor)
}

func parseSingleValueForChanged(tyChanged reflect.Type) parsedQuery {
	assertIsNonPointerType(tyChanged)

	// instantiate a new instance in memory. we do that to get access
	// to the inner type
	rAccessor := ptrValueOf(reflect.New(tyChanged))
	accessor := rAccessor.Interface().(changedAccessor)
	innerType := reflectComponentTypeOf(inner.TypeOf(accessor))

	// build a query for the inner value
	innerQuery := buildQuerySingleValue(innerType.Type)

	// mark types as mutable if needed
	var mutableComponentTypes []ComponentType
	if accessor.mutable() {
		mutableComponentTypes = append(mutableComponentTypes, innerType)
	}

	return parsedQuery{
		mutableComponentTypes: mutableComponentTypes,

		hasValue: func(world *World, system *preparedSystem, entity *Entity) bool {
			// check the entity for the component
			componentValue, ok := entity.Components[innerType]
			if !ok {
				return false
			}

			return componentValue.LastChanged >= system.LastRun
		},

		putValue: func(entity *Entity, target reflect.Value) bool {
			// target should point to an Changed[X]
			assertIsNonPointerType(target.Type())

			accessor := target.Addr().Interface().(changedAccessor)

			innerValue := accessor.innerValue()
			ok := innerQuery.putValue(entity, innerValue)
			if !ok {
				return false
			}

			return true
		},
	}
}

func assertIsComponentType(ty reflect.Type) {
	if !isComponentType(ty) {
		panic(fmt.Sprintf("expected component type, got %s", ty))
	}
}
