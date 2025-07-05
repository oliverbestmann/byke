package byke

import "reflect"

// Local provides a value local to the system.
// It must be injected into a system as a pointer.
type Local[T any] struct {
	Value T
}

func (l *Local[T]) init(*World) SystemParamState {
	return l
}

func (l *Local[T]) getValue(*preparedSystem) reflect.Value {
	return reflect.ValueOf(l)
}

func (l *Local[T]) cleanupValue(reflect.Value) {
	// no cleanup needed
}

func (*Local[T]) valueType() reflect.Type {
	return reflect.TypeFor[*Local[T]]()
}
