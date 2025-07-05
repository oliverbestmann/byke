package byke

import "reflect"

func Type[T any]() TypeRef[T] {
	return TypeRef[T]{}
}

// TypeRef provides an easy workaround to the lack of proper support for
// type parameters in go reflection. Just embed an instance of the zero
// sized type TypeRef into your struct and parameterize it with the generic
// parameter you want to query later.
type TypeRef[S any] struct{}

func (TypeRef[S]) ReflectType() reflect.Type {
	return reflect.TypeFor[S]()
}

type HasType interface {
	ReflectType() reflect.Type
}
