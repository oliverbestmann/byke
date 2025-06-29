package ecs

import "reflect"

type Has[C IsComponent[C]] struct {
	InnerType[C]
	value bool
}

func (h *Has[C]) setValue(value bool) {
	h.value = value
}

func (h *Has[C]) isHasTypeMarker(accessor hasAccessor) {}

func (h *Has[C]) Exists() bool {
	return h.value
}

type hasAccessor interface {
	innerType() reflect.Type
	setValue(bool)
	isHasTypeMarker(hasAccessor)
}

func isHasType(tyTarget reflect.Type) bool {
	tyOptionAccessor := reflect.TypeFor[hasAccessor]()

	return tyTarget.Kind() != reflect.Pointer &&
		reflect.PointerTo(tyTarget).Implements(tyOptionAccessor)
}

func parseSingleValueForHas(tyHas reflect.Type) queryValueAccessor {
	assertIsNonPointerType(tyHas)

	// instantiate a new option in memory
	ptrToHas := pointerValue{Value: reflect.New(tyHas)}

	// get the accessor
	accessor := ptrToHas.Interface().(hasAccessor)

	// get an extractor for the inner type
	extractor := extractComponentByType(reflectComponentTypeOf(accessor.innerType()))

	return queryValueAccessor{
		extractor: func(entity *Entity) (pointerValue, bool) {
			_, hasValue := extractor(entity)
			accessor.setValue(hasValue)
			return ptrToHas, true
		},

		populateTarget: func(target reflect.Value, ptrToValue pointerValue) {
			assertIsNonPointerType(target.Type())
			assertIsPointerType(ptrToValue.Type())

			target.Set(ptrToValue.Elem())
		},
	}
}
