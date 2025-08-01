package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/refl"
	spoke "github.com/oliverbestmann/byke/spoke"
	"math"
	"reflect"
	"unsafe"
)

type ParsedQuery struct {
	Builder spoke.QueryBuilder
	Mutable []*spoke.ComponentType
	Setters []Setter
}

type SetValue func(target any, ref spoke.EntityRef)

type Setter struct {
	// offset of this field to the start of the struct
	UnsafeFieldOffset uintptr

	ComponentIdx      int
	ComponentTypeSize uintptr

	UnsafeCopyComponentValue bool
	UnsafeCopyComponentAddr  bool

	// UseEntityId implies that UnsafeFieldOffset is a pointer to an EntityId variable
	// and we're supposed to copy the value of the current EntityId into that field.
	UseEntityId bool
}

func FromEntity[T any](setters []Setter, ref spoke.EntityRef) T {
	var target T

	ptrToTarget := unsafe.Pointer(&target)

	for idx := range setters {
		setter := &setters[idx]

		switch {
		case setter.UnsafeCopyComponentValue:
			// target points to a component value
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			source := ref.GetAt(setter.ComponentIdx)

			if source != nil {
				// I think this is safe as we're copying a value from the heap to the stack,
				// and we do not need a write barrier in that case.
				memmove(target, source, setter.ComponentTypeSize)
			} else {
				// no value, clear memory
				clear((*buf(target))[:setter.ComponentTypeSize])
			}

		case setter.UnsafeCopyComponentAddr:
			// target points to a pointer to a component value
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			source := ref.GetAt(setter.ComponentIdx)

			// set the target pointer to the address of the source
			*(*unsafe.Pointer)(target) = source

		case setter.UseEntityId:
			target := unsafe.Add(ptrToTarget, setter.UnsafeFieldOffset)
			*(*spoke.EntityId)(target) = ref.EntityId()
		}
	}

	return target
}

func ParseQuery(queryType reflect.Type) (ParsedQuery, error) {
	var parsed ParsedQuery

	if err := buildQuery(queryType, &parsed, nil, 0); err != nil {
		return ParsedQuery{}, err
	}

	return parsed, nil
}

func buildQuery(queryType reflect.Type, result *ParsedQuery, path []int, offset uintptr) error {
	query := &result.Builder

	switch {
	case isEntityId(queryType):
		result.Setters = append(result.Setters, Setter{
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
		query.Filter(filter.applyTo(result, offset))

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
	return ty.Kind() != reflect.Pointer && ty.Implements(reflect.TypeFor[spoke.IsErasedImmutableComponent]())
}

func isFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && refl.ImplementsInterfaceDirectly[Filter](ty)
}

func isEmbeddableFilter(ty reflect.Type) bool {
	return ty.Kind() != reflect.Pointer && refl.ImplementsInterfaceDirectly[EmbeddableFilter](ty)
}

func isEntityId(ty reflect.Type) bool {
	return ty == reflect.TypeFor[spoke.EntityId]()
}

type buf *[math.MaxInt32]byte

func memmove(target, source unsafe.Pointer, byteCount uintptr) {
	copy((*buf(target))[:byteCount], (*buf(source))[:byteCount])
}
