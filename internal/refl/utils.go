package refl

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"iter"
	"reflect"
)

func ComponentTypeOf(ty reflect.Type) *arch.ComponentType {
	if !IsComponent(ty) {
		panic(fmt.Sprintf("type %s is not a component", ty))
	}

	component := reflect.New(ty).Interface().(arch.ErasedComponent)
	return component.ComponentType()
}

func IterFields(ty reflect.Type) iter.Seq[reflect.StructField] {
	return func(yield func(reflect.StructField) bool) {
		for idx := range ty.NumField() {
			if !yield(ty.Field(idx)) {
				return
			}
		}
	}
}

func ImplementsInterfaceDirectly[If any](ty reflect.Type) bool {
	iface := reflect.TypeFor[If]()

	if !ty.Implements(iface) {
		return false
	}

	for ty.Kind() == reflect.Pointer {
		ty = ty.Elem()
	}

	for field := range IterFields(ty) {
		if !field.Anonymous {
			continue
		}

		if field.Type.Implements(iface) {
			return false
		}

		if reflect.PointerTo(field.Type).Implements(iface) {
			return false
		}
	}

	return true
}

func IsComponent(ty reflect.Type) bool {
	if ty.Kind() != reflect.Struct {
		return false
	}

	if !ty.Implements(reflect.TypeFor[arch.ErasedComponent]()) {
		return false
	}

	// a component must embed arch.Component or arch.ComparableComponent
	var count int
	for field := range IterFields(ty) {
		if ImplementsInterfaceDirectly[arch.ErasedComponent](field.Type) {
			count += 1
		}
	}

	// expect to have exactly one
	return count == 1
}
