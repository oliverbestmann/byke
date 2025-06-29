package byke

import (
	"fmt"
	"reflect"
)

type optionAccessor interface {
	__isOption()
	innerType() reflect.Type
	setValue(value any)
	mutable() bool
}

type Option[T IsComponent[T]] struct {
	InnerType[T]
	value *T
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

func (o *Option[T]) setValue(value any) {
	if value == nil {
		o.value = nil
	} else {
		o.value = value.(*T)
	}
}

type OptionMut[T IsComponent[T]] struct {
	InnerType[T]
	value *T
}

func (o *OptionMut[T]) mutable() bool {
	return true
}

func (o *OptionMut[T]) Get() (*T, bool) {
	return o.value, o.value != nil
}

func (o *OptionMut[T]) __isOption() {}

func (o *OptionMut[T]) setValue(value any) {
	if value == nil {
		o.value = nil
	} else {
		o.value = value.(*T)
	}
}

func isOptionType(tyTarget reflect.Type) bool {
	tyOptionAccessor := reflect.TypeFor[optionAccessor]()

	fmt.Println(tyTarget, reflect.PointerTo(tyTarget).Implements(tyOptionAccessor))

	return tyTarget.Kind() != reflect.Pointer &&
		reflect.PointerTo(tyTarget).Implements(tyOptionAccessor)
}

func parseSingleValueForOption(tyOption reflect.Type) queryValueAccessor {
	assertIsNonPointerType(tyOption)

	// instantiate a new option in memory
	ptrToOption := pointerValue{Value: reflect.New(tyOption)}

	// get the accessor
	accessor := ptrToOption.Interface().(optionAccessor)

	// get an extractor for the inner type
	extractor := extractComponentByType(reflectComponentTypeOf(accessor.innerType()))

	return queryValueAccessor{
		extractor: func(entity *Entity) (pointerValue, bool) {
			ptrToValue, hasValue := extractor(entity)

			if !hasValue {
				accessor.setValue(nil)
				return ptrToOption, true
			}

			// set the actual value in the option
			accessor.setValue(ptrToValue.Interface())
			return ptrToOption, true
		},

		populateTarget: func(target reflect.Value, ptrToValue pointerValue) {
			assertIsNonPointerType(target.Type())
			assertIsPointerType(ptrToValue.Type())

			target.Set(ptrToValue.Elem())
		},
	}
}
