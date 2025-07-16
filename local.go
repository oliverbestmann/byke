package byke

import "reflect"

// Local is a SystemParam that provides a value local to a system.
// It must be injected as a pointer value.
//
// A system can have multiple independent Local parameters even with the same type T.
type Local[T any] struct {
	_     noCopy
	Value T
}

func (l *Local[T]) init(*World) SystemParamState {
	return l
}

func (l *Local[T]) getValue(systemContext) reflect.Value {
	return reflect.ValueOf(l)
}

func (l *Local[T]) cleanupValue() {}

func (*Local[T]) valueType() reflect.Type {
	return reflect.TypeFor[*Local[T]]()
}
