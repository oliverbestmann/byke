package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/assert"
	"reflect"
	"slices"
)

type ParsedQuery struct {
	Query   arch.Query
	Mutable []*arch.ComponentType
	Setters []Setter
}

type SetValue func(target any, ref arch.EntityRef)

type Setter struct {
	Field    []int
	SetValue SetValue
}

func FromEntity(target any, setters []Setter, ref arch.EntityRef) {
	rValue := reflect.ValueOf(target)
	assert.IsPointerType(rValue.Type())

	for _, setter := range setters {
		target := rValue
		if setter.Field != nil {
			// rValue must be a pointer to a struct
			target = rValue.Elem().FieldByIndex(setter.Field).Addr()
		}

		setter.SetValue(target.Interface(), ref)
	}
}

func ParseQuery(queryType reflect.Type) (ParsedQuery, error) {
	var parsed ParsedQuery

	if err := buildQuery(queryType, &parsed, nil); err != nil {
		return ParsedQuery{}, err
	}

	return parsed, nil
}

func buildQuery(queryType reflect.Type, result *ParsedQuery, path []int) error {
	query := &result.Query

	switch {
	case isEntityId(queryType):
		result.Setters = append(result.Setters, Setter{
			Field: slices.Clone(path),
			SetValue: func(target any, ref arch.EntityRef) {
				*target.(*arch.EntityId) = ref.EntityId
			},
		})

		return nil

	case isComponent(queryType):
		componentType := componentTypeOf(queryType)
		query.Fetch = append(query.Fetch, componentType)

		result.Setters = append(result.Setters, Setter{
			Field: slices.Clone(path),
			SetValue: func(target any, ref arch.EntityRef) {
				value, ok := ref.Get(componentType)
				if !ok {
					panic(fmt.Sprintf("entity does not contain component: %s", componentType))
				}

				// target is a pointer to the component value
				componentType.SetValue(target.(arch.ErasedComponent), value.Value)
			},
		})

		return nil

	case isMutableComponent(queryType):
		componentType := componentTypeOf(queryType.Elem())
		query.Fetch = append(query.Fetch, componentType)
		result.Mutable = append(result.Mutable, componentType)

		result.Setters = append(result.Setters, Setter{
			Field: slices.Clone(path),
			SetValue: func(target any, ref arch.EntityRef) {
				value, ok := ref.Get(componentType)
				if !ok {
					panic(fmt.Sprintf("entity does not contain component: %s", componentType))
				}

				// target is a pointer to a pointer the component value
				componentType.SetValue(target, value.Value)
			},
		})

		return nil

	case isFilter(queryType):
		filter := reflect.New(queryType).Interface().(Filter)
		filter.applyTo(result)

		result.Setters = append(result.Setters, Setter{
			Field: slices.Clone(path),
			SetValue: func(target any, ref arch.EntityRef) {
				target.(FromEntityRef).FromEntityRef(ref)
			},
		})

		return nil

	case isStructQuery(queryType):
		return buildStructQuery(queryType, result, path)

	default:
		return fmt.Errorf("invalid query type: %s", queryType)
	}
}

func buildStructQuery(queryType reflect.Type, result *ParsedQuery, path []int) error {
	for field := range fieldsOf(queryType) {
		if field.Anonymous {
			allowed := isEmbeddableFilter(field.Type) || isEntityId(field.Type)
			if !allowed {
				return fmt.Errorf("must not be embedded in query target %s: %s", queryType, field.Type)
			}
		}

		pathToField := append(path, field.Index...)
		if err := buildQuery(field.Type, result, pathToField); err != nil {
			return err
		}
	}

	return nil
}

func isStructQuery(ty reflect.Type) bool {
	return ty.Kind() == reflect.Struct
}

func isComponent(ty reflect.Type) bool {
	if ty.Kind() != reflect.Struct {
		return false
	}

	if !ty.Implements(reflect.TypeFor[arch.ErasedComponent]()) {
		return false
	}

	// a component must embed arch.Component or arch.ComparableComponent
	var count int
	for field := range fieldsOf(ty) {
		if implementsInterfaceDirectly[arch.ErasedComponent](field.Type) {
			count += 1
		}
	}

	// expect to have exactly one
	return count == 1
}

func isMutableComponent(ty reflect.Type) bool {
	return ty.Kind() == reflect.Pointer && isComponent(ty.Elem())
}

func isFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && implementsInterfaceDirectly[Filter](reflect.PointerTo(ty))
}

func isEmbeddableFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && implementsInterfaceDirectly[EmbeddableFilter](ty)
}

func isEntityId(ty reflect.Type) bool {
	return ty == reflect.TypeFor[arch.EntityId]()
}
