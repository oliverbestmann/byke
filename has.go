package byke

import (
	"github.com/oliverbestmann/byke/internal/inner"
	"reflect"
)

type Has[C IsComponent[C]] struct {
	inner.Type[C]
	value bool
}

func (h *Has[C]) setValue(value bool) {
	h.value = value
}

func (h *Has[C]) isHasTypeMarker(hasAccessor) {}

func (h *Has[C]) Exists() bool {
	return h.value
}

type hasAccessor interface {
	inner.HasType
	setValue(bool)
	isHasTypeMarker(hasAccessor)
}

func isHasType(tyTarget reflect.Type) bool {
	tyOptionAccessor := reflect.TypeFor[hasAccessor]()

	return tyTarget.Kind() != reflect.Pointer &&
		reflect.PointerTo(tyTarget).Implements(tyOptionAccessor)
}

func parseSingleValueForHas(tyHas reflect.Type) parsedQuery {
	assertIsNonPointerType(tyHas)

	// instantiate a new option in memory. we do that to get access
	// to the tyHas' inner type
	ptrToHas := ptrValueOf(reflect.New(tyHas))
	accessor := ptrToHas.Interface().(hasAccessor)
	innerType := inner.TypeOf(accessor)

	scratch := ptrValueOf(reflect.New(innerType))
	innerQuery := buildQuerySingleValue(innerType)

	return parsedQuery{
		putValue: func(entity *Entity, target reflect.Value) bool {
			// target should point to an Has[X]
			assertIsNonPointerType(target.Type())

			accessor := target.Addr().Interface().(hasAccessor)

			ok := innerQuery.putValue(entity, scratch.Elem())
			accessor.setValue(ok)

			return true
		},
	}
}
