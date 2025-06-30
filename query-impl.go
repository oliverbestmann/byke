package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/inner"
	"github.com/oliverbestmann/byke/internal/set"
	"reflect"
	"slices"
)

type PopulateTarget func(target reflect.Value, ptrToValues []ptrValue)

type PopulateSingleTarget func(target reflect.Value, ptrToValues ptrValue)

type erasedQuery struct {
	world  *World
	parsed parsedQuery
}

type queryAccessor interface {
	inner.HasType
	isQuery(queryAccessor)
	set(query *erasedQuery)
	get() *erasedQuery
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
	parsed := buildQueryTarget(targetType)

	// set the backend of the query that performs the actual
	// generic query work
	queryAcc.set(&erasedQuery{world: w, parsed: parsed})

	// return the Query[C] instance
	return ptrToQuery.Elem()
}

// parsedQuery extracts a value from an entity and
// puts them into a target value
type parsedQuery struct {
	putValue func(entity *Entity, target reflect.Value) bool
	hasValue func(entity *Entity) bool

	// slice that identifies the mutable component types
	// that this parsedQuery will extract.
	mutableComponentTypes []ComponentType
}

func ptrToComponentValue(entity *Entity, ty ComponentType) (AnyComponent, bool) {
	value, ok := entity.Components[ty]
	return value.PtrToValue, ok
}

func buildQueryTarget(tyTarget reflect.Type) parsedQuery {
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

func parseStructQueryTarget(tyTarget reflect.Type) parsedQuery {
	var parsedQueries []parsedQuery

	var mutableComponentTypes set.Set[ComponentType]

	for idx := range tyTarget.NumField() {
		field := tyTarget.Field(idx)
		fieldTy := field.Type

		if !field.IsExported() || field.Anonymous {
			continue
		}

		delegate := buildQuerySingleValue(fieldTy)

		parsedQueries = append(parsedQueries, parsedQuery{
			hasValue: delegate.hasValue,

			putValue: func(entity *Entity, target reflect.Value) bool {
				fieldTarget := target.Field(idx)
				return delegate.putValue(entity, fieldTarget)
			},
		})

		for _, ty := range delegate.mutableComponentTypes {
			mutableComponentTypes.Insert(ty)
		}
	}

	return parsedQuery{
		mutableComponentTypes: slices.Collect(mutableComponentTypes.Values()),

		hasValue: func(entity *Entity) bool {
			for _, ex := range parsedQueries {
				if ex.hasValue != nil && !ex.hasValue(entity) {
					return false
				}
			}

			return true
		},

		putValue: func(entity *Entity, target reflect.Value) bool {
			for _, ex := range parsedQueries {
				if !ex.putValue(entity, target) {
					return false
				}
			}

			return true
		},
	}
}

// entityIdQuery is an parsedQuery that extracts the entity id of an Entity
var entityIdQuery = parsedQuery{
	putValue: func(entity *Entity, target reflect.Value) bool {
		assertIsNonPointerType(target.Type())
		target.Set(reflect.ValueOf(&entity.Id).Elem())
		return true
	},
}

func buildQuerySingleValue(tyTarget reflect.Type) parsedQuery {
	switch {
	// the entity id is directly injectable
	case tyTarget == reflect.TypeFor[EntityId]():
		return entityIdQuery

	case isPointerToParentComponentType(tyTarget):
		panic(fmt.Sprintf("parent side of relation must not be queried via pointer: %s", tyTarget))

	case isComponentType(tyTarget):
		componentType := reflectComponentTypeOf(tyTarget)
		return parsedQuery{
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

		return parsedQuery{
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

			mutableComponentTypes: []ComponentType{componentType},
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
