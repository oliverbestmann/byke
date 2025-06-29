package inner

import "reflect"

// Type provides an easy workaround to the lack of proper support for
// type parameters in go reflection. Just embed an instance of the zero
// sized type Type into your struct and parameterize it with the generic
// parameter you want to query later.
type Type[S any] struct{}

func (Type[S]) innerType() reflect.Type {
	return reflect.TypeFor[S]()
}

type HasType interface {
	innerType() reflect.Type
}

func TypeOf(inner HasType) reflect.Type {
	return inner.innerType()
}
