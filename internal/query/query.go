package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/refl"
	"math"
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

type Setter struct {
	Field    []int
	SetValue SetValue

	// offset of this field to the start of the struct
	UnsafeFieldOffset uintptr

	// UseEntityId implies that UnsafeFieldOffset is a pointer to an EntityId variable
	// and we're supposed to copy the value of the current EntityId into that field.
	UseEntityId bool

	UnsafeCopyComponentValue bool
	UnsafeCopyComponentAddr  bool

	ComponentIdx      int
	ComponentTypeSize uintptr
}

func FromEntity[T any](target *T, setters []Setter, ref arch.EntityRef) {
	ptrToTarget := unsafe.Pointer(target)

	for idx := range setters {
		setter := &setters[idx]

		if setter.UnsafeCopyComponentValue {
			// target points to a component value
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			source := ref.GetAt(setter.ComponentIdx)

			if source != nil {
				memmove(target, source, setter.ComponentTypeSize)
			} else {
				// no value, clear memory
				clear((*buf(target))[:setter.ComponentTypeSize])
			}

			continue
		}

		if setter.UnsafeCopyComponentAddr {
			// target points to a pointer to a component value
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			source := ref.GetAt(setter.ComponentIdx)

			// set the target pointer to the address of the source
			*(*unsafe.Pointer)(target) = source

			continue
		}

		if setter.UseEntityId {
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			*(*arch.EntityId)(target) = ref.EntityId()
			continue
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
			Field:             slices.Clone(path),
			UnsafeFieldOffset: offset,
			UseEntityId:       true,
		})

		return nil

	case refl.IsComponent(queryType):
		componentType := refl.ComponentTypeOf(queryType)
		componentIdx := query.FetchComponent(componentType, false)

		result.Setters = append(result.Setters, Setter{
			UnsafeFieldOffset: offset,

			UnsafeCopyComponentValue: true,
			ComponentTypeSize:        componentType.Type.Size(),
			ComponentIdx:             componentIdx,
		})

		return nil

	case isMutableComponent(queryType):
		if isImmutableComponent(queryType.Elem()) {
			panic(fmt.Sprintf("Can not inject pointer to ImmutableComponent %s", queryType.Elem()))
		}

		componentType := refl.ComponentTypeOf(queryType.Elem())
		componentIdx := query.FetchComponent(componentType, false)
		result.Mutable = append(result.Mutable, componentType)

		result.Setters = append(result.Setters, Setter{
			UnsafeFieldOffset: offset,

			UnsafeCopyComponentAddr: true,
			ComponentIdx:            componentIdx,
		})

		return nil

	case isFilter(queryType):
		filter := reflect.New(queryType).Interface().(Filter)
		filters := filter.applyTo(result, offset)

		// calculate the filters and add them to the query
		query.Filters = append(query.Filters, filters...)

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

type buf *[math.MaxInt32]byte

func memmove(target, source unsafe.Pointer, count uintptr) {
	copy((*buf(target))[:count], (*buf(source))[:count])
}
