package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"iter"
	"reflect"
)

func componentTypeOf(ty reflect.Type) *arch.ComponentType {
	if !isComponent(ty) {
		panic(fmt.Sprintf("type %s is not a component", ty))
	}

	component := reflect.New(ty).Interface().(arch.ErasedComponent)
	return component.ComponentType()
}

func fieldsOf(ty reflect.Type) iter.Seq[reflect.StructField] {
	return func(yield func(reflect.StructField) bool) {
		for idx := range ty.NumField() {
			if !yield(ty.Field(idx)) {
				return
			}
		}
	}
}

func implementsInterfaceDirectly[If any](ty reflect.Type) bool {
	iface := reflect.TypeFor[If]()

	if !ty.Implements(iface) {
		return false
	}

	for ty.Kind() == reflect.Pointer {
		ty = ty.Elem()
	}

	for field := range fieldsOf(ty) {
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
