package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/inner"
	"reflect"
)

type PopulateTarget func(target reflect.Value, ptrToValues []ptrValue)

type PopulateSingleTarget func(target reflect.Value, ptrToValues ptrValue)

type erasedQuery struct {
	world     *World
	extractor extractor
}

type queryAccessor interface {
	inner.HasType
	isQuery(queryAccessor)
	set(query *erasedQuery)
}

// ensure Query implements the queryAccessor type
var _ queryAccessor = &Query[any]{}

// buildQuery parses the Query[C] type into a reflect.Value
// holding an actual Query[C] instance.
func buildQuery(w *World, queryType reflect.Type) reflect.Value {
	// allocate a new Query object in memory
	var ptrToQuery = reflect.New(queryType)
	queryAcc := ptrToQuery.Interface().(queryAccessor)

	// build the query from the target type
	targetType := inner.TypeOf(queryAcc)
	extractor := buildQueryTarget(targetType)

	// set the backend of the query that performs the actual
	// generic query work
	queryAcc.set(&erasedQuery{world: w, extractor: extractor})

	// return the Query[C] instance
	return ptrToQuery.Elem()
}

// extractor extracts a value from an entity and
// puts them into a target value
type extractor struct {
	putValue func(entity *Entity, target reflect.Value) bool
	hasValue func(entity *Entity) bool
}

func ptrToComponentValue(entity *Entity, ty ComponentType) (AnyComponent, bool) {
	value, ok := entity.Components[ty]
	return value.PtrToValue, ok
}

func buildQueryTarget(tyTarget reflect.Type) extractor {
	isSingleTarget := isComponentType(tyTarget) ||
		tyTarget.Kind() == reflect.Pointer && isComponentType(tyTarget.Elem()) ||
		isOptionType(tyTarget)

	if isSingleTarget {
		return buildQuerySingleValue(tyTarget)
	}

	if tyTarget.Kind() == reflect.Struct {
		return parseStructQueryTarget(tyTarget)
	}

	panic(fmt.Sprintf("unknown query target type: %s", tyTarget))
}

func assertIsPointerType(t reflect.Type) {
	if t.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("expected pointer type, got %s", t))
	}
}

func assertIsNonPointerType(t reflect.Type) {
	if t.Kind() == reflect.Pointer {
		panic(fmt.Sprintf("expected non pointer type, got %s", t))
	}
}

func parseStructQueryTarget(tyTarget reflect.Type) extractor {
	var extractors []extractor

	for idx := range tyTarget.NumField() {
		field := tyTarget.Field(idx)
		fieldTy := field.Type

		if !field.IsExported() || field.Anonymous {
			continue
		}

		delegate := buildQuerySingleValue(fieldTy)

		extractors = append(extractors, extractor{
			hasValue: delegate.hasValue,

			putValue: func(entity *Entity, target reflect.Value) bool {
				fieldTarget := target.Field(idx)
				return delegate.putValue(entity, fieldTarget)
			},
		})
	}

	return extractor{
		hasValue: func(entity *Entity) bool {
			for _, ex := range extractors {
				if ex.hasValue != nil && !ex.hasValue(entity) {
					return false
				}
			}

			return true
		},

		putValue: func(entity *Entity, target reflect.Value) bool {
			for _, ex := range extractors {
				if !ex.putValue(entity, target) {
					return false
				}
			}

			return true
		},
	}
}

// entityIdExtractor is an extractor that extracts the entity id of an Entity
var entityIdExtractor = extractor{
	putValue: func(entity *Entity, target reflect.Value) bool {
		assertIsNonPointerType(target.Type())
		target.Set(reflect.ValueOf(&entity.Id).Elem())
		return true
	},
}

func buildQuerySingleValue(tyTarget reflect.Type) extractor {
	switch {
	// the entity id is directly injectable
	case tyTarget == reflect.TypeFor[EntityId]():
		return entityIdExtractor

	case isPointerToParentComponentType(tyTarget):
		panic(fmt.Sprintf("parent side of relation must not be queried via pointer: %s", tyTarget))

	case isComponentType(tyTarget):
		componentType := reflectComponentTypeOf(tyTarget)
		return extractor{
			hasValue: func(entity *Entity) bool {
				_, ok := ptrToComponentValue(entity, componentType)
				return ok
			},

			putValue: func(entity *Entity, target reflect.Value) bool {
				assertIsNonPointerType(target.Type())

				value, ok := ptrToComponentValue(entity, componentType)
				if !ok {
					return false
				}

				target.Set(reflect.ValueOf(value).Elem())
				return true
			},
		}

	case tyTarget.Kind() == reflect.Pointer && isComponentType(tyTarget.Elem()):
		componentType := reflectComponentTypeOf(tyTarget.Elem())

		return extractor{
			hasValue: func(entity *Entity) bool {
				_, ok := ptrToComponentValue(entity, componentType)
				return ok
			},

			putValue: func(entity *Entity, target reflect.Value) bool {
				assertIsPointerType(target.Type())

				value, ok := ptrToComponentValue(entity, componentType)
				if !ok {
					return false
				}

				// let target point to the value
				target.Set(reflect.ValueOf(value))
				return true
			},
		}

	case isOptionType(tyTarget):
		return parseSingleValueForOption(tyTarget)

	case isHasType(tyTarget):
		return parseSingleValueForHas(tyTarget)

	default:
		panic(fmt.Sprintf("not a type we can extract: %s", tyTarget))
	}
}

func isPointerToParentComponentType(target reflect.Type) bool {
	if target.Kind() != reflect.Pointer {
		return false
	}

	return target.Implements(reflect.TypeFor[parentComponent]())
}

func isComponentType(t reflect.Type) bool {
	return t.Kind() != reflect.Pointer && t.Implements(reflect.TypeFor[AnyComponent]())
}
