package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/refl"
	"reflect"
	"slices"
	"unsafe"
)

type ParsedQuery struct {
	Query   arch.Query
	Mutable []*arch.ComponentType
	Setters []Setter
}

type SetValue func(target any, ref arch.EntityRef)

type UnsafeSetValue func(target unsafe.Pointer, ref arch.EntityRef)

type Setter struct {
	Field    []int
	SetValue SetValue

	// offset of this field to the start of the struct
	UnsafeFieldOffset uintptr

	// Type of the field
	UnsafeSetValue UnsafeSetValue
}

func FromEntity[T any](target *T, setters []Setter, ref arch.EntityRef) {
	ptrToTarget := unsafe.Pointer(target)

	for _, setter := range setters {
		if setter.UnsafeSetValue != nil {
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			setter.UnsafeSetValue(target, ref)
		}

		if setter.SetValue != nil {
			target := reflect.ValueOf(target)

			if setter.Field != nil {
				// let target point to a field within the target struct
				target = target.Elem().FieldByIndex(setter.Field).Addr()
			}

			setter.SetValue(target.Interface(), ref)
		}
	}
}

func ParseQuery(queryType reflect.Type) (ParsedQuery, error) {
	var parsed ParsedQuery

	if err := buildQuery(queryType, &parsed, nil, 0); err != nil {
		return ParsedQuery{}, err
	}

	return parsed, nil
}

func buildQuery(queryType reflect.Type, result *ParsedQuery, path []int, offset uintptr) error {
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

	case refl.IsComponent(queryType):
		componentType := refl.ComponentTypeOf(queryType)
		query.Fetch = append(query.Fetch, componentType)

		result.Setters = append(result.Setters, Setter{
			UnsafeFieldOffset: offset,
			UnsafeSetValue: func(target unsafe.Pointer, ref arch.EntityRef) {
				value := ref.Get(componentType)
				if value == nil {
					panic(fmt.Sprintf("entity does not contain component: %s", componentType))
				}

				componentType.UnsafeSetValue(target, value)
			},
		})

		return nil

	case isMutableComponent(queryType):
		if isImmutableComponent(queryType.Elem()) {
			panic(fmt.Sprintf("Can not inject pointer to ImmutableComponent %s", queryType.Elem()))
		}

		componentType := refl.ComponentTypeOf(queryType.Elem())
		query.Fetch = append(query.Fetch, componentType)
		result.Mutable = append(result.Mutable, componentType)

		result.Setters = append(result.Setters, Setter{
			UnsafeFieldOffset: offset,
			UnsafeSetValue: func(target unsafe.Pointer, ref arch.EntityRef) {
				value := ref.Get(componentType)
				if value == nil {
					panic(fmt.Sprintf("entity does not contain component: %s", componentType))
				}

				componentType.UnsafeSetPointer(target, value)
			},
		})

		return nil

	case isFilter(queryType):
		filter := reflect.New(queryType).Interface().(Filter)

		// calculate the filters and add them to the query
		query.Filters = append(query.Filters, filter.applyTo(result)...)

		if isFromEntityRef(queryType) {
			result.Setters = append(result.Setters, Setter{
				Field: slices.Clone(path),
				SetValue: func(target any, ref arch.EntityRef) {
					target.(FromEntityRef).fromEntityRef(ref)
				},
			})
		}

		return nil

	case isStructQuery(queryType):
		return buildStructQuery(queryType, result, path, offset)

	default:
		return fmt.Errorf("invalid query type: %s", queryType)
	}
}

func buildStructQuery(queryType reflect.Type, result *ParsedQuery, path []int, baseOffset uintptr) error {
	for field := range refl.IterFields(queryType) {
		if field.Anonymous {
			allowed := isEmbeddableFilter(field.Type) || isEntityId(field.Type)
			if !allowed {
				return fmt.Errorf("must not be embedded in query target %s: %s", queryType, field.Type)
			}
		}

		offset := baseOffset + field.Offset
		pathToField := append(path, field.Index...)
		if err := buildQuery(field.Type, result, pathToField, offset); err != nil {
			return err
		}
	}

	return nil
}

func isStructQuery(ty reflect.Type) bool {
	return ty.Kind() == reflect.Struct
}

func isMutableComponent(ty reflect.Type) bool {
	return ty.Kind() == reflect.Pointer && refl.IsComponent(ty.Elem())
}

func isImmutableComponent(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && ty.Implements(reflect.TypeFor[arch.IsErasedImmutableComponent]())
}

func isFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && refl.ImplementsInterfaceDirectly[Filter](ty)
}

func isEmbeddableFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && refl.ImplementsInterfaceDirectly[EmbeddableFilter](ty)
}

func isFromEntityRef(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && refl.ImplementsInterfaceDirectly[FromEntityRef](reflect.PointerTo(ty))
}

func isEntityId(ty reflect.Type) bool {
	return ty == reflect.TypeFor[arch.EntityId]()
}
