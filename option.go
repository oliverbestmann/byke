package byke

import (
	"github.com/oliverbestmann/byke/internal/inner"
	"reflect"
)

type optionAccessor interface {
	inner.HasType
	__isOption()
	mutable() bool
	ptrInner() ptrValue
}

type Option[T IsComponent[T]] struct {
	inner.Type[T]
	value *T
}

func (o *Option[T]) ptrInner() ptrValue {
	return ptrValueOf(reflect.ValueOf(&o.value).Elem())
}

func (o *Option[T]) mutable() bool {
	return false
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

type OptionMut[T IsComponent[T]] struct {
	inner.Type[T]
	value *T
}

func (o *OptionMut[T]) mutable() bool {
	return true
}

func (o *OptionMut[T]) Get() (*T, bool) {
	return o.value, o.value != nil
}

func (o *OptionMut[T]) __isOption() {}

func (o *OptionMut[T]) ptrInner() ptrValue {
	return ptrValueOf(reflect.ValueOf(&o.value).Elem())
}

func isOptionType(tyTarget reflect.Type) bool {
	tyOptionAccessor := reflect.TypeFor[optionAccessor]()

	return tyTarget.Kind() != reflect.Pointer &&
		reflect.PointerTo(tyTarget).Implements(tyOptionAccessor)
}

func parseSingleValueForOption(tyOption reflect.Type) parsedQuery {
	assertIsNonPointerType(tyOption)

	// instantiate a new option in memory. we do that to get access
	// to the tyOptions inner type
	ptrToOption := ptrValueOf(reflect.New(tyOption))
	accessor := ptrToOption.Interface().(optionAccessor)
	innerType := inner.TypeOf(accessor)

	innerQuery := buildQuerySingleValue(reflect.PointerTo(innerType))

	var mutableComponentTypes []ComponentType
	if accessor.mutable() {
		mutableComponentTypes = append(mutableComponentTypes, reflectComponentTypeOf(innerType))
	}

	return parsedQuery{
		mutableComponentTypes: mutableComponentTypes,

		putValue: func(entity *Entity, target reflect.Value) bool {
			// target should point to an Option[X]
			assertIsNonPointerType(target.Type())

			accessor := target.Addr().Interface().(optionAccessor)

			ptrInner := accessor.ptrInner()
			ok := innerQuery.putValue(entity, ptrInner.Value)
			if !ok {
				// set pointer nil
				ptrInner.Set(reflect.Zero(reflect.PointerTo(innerType)))
			}

			return true
		},
	}
}
