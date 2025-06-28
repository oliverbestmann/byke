package ecs

import (
	"reflect"
)

type optionAccessor interface {
	__isOption()
	reflectType() reflect.Type
	setValue(value any)
}

type Option[T IsComponent[T]] struct {
	value *T
}

func (o *Option[T]) Get() (T, bool) {
	return o.OrDefault(), o.value != nil
}

func (o *Option[T]) OrValue(fallback T) T {
	if o.value != nil {
		return *o.value
	}

	return fallback
}

func (o *Option[T]) OrDefault() T {
	var tZero T
	return o.OrValue(tZero)
}

func (o *Option[T]) __isOption() {}

func (o *Option[T]) reflectType() reflect.Type {
	return reflect.TypeFor[T]()
}

func (o *Option[T]) setValue(value any) {
	o.value = value.(*T)
}

func extractOptionOf(tyTarget reflect.Type) Extractor {
	// tyTarget is of type Option[xx]

	option := pointerValueOf(reflect.New(tyTarget))

	return func(entity *Entity) (pointerValue, bool) {
		accessor := option.Interface().(optionAccessor)

		tyComponent := reflectComponentTypeOf(accessor.reflectType())
		extractor := extractComponentByType(tyComponent)

		value, ok := extractor(entity)
		if ok {
			accessor.setValue(value.Interface())
		}

		return option, true
	}
}
