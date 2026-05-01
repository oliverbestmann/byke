package byke

import (
	"reflect"
)

// Local is a SystemParam that provides a value local to a system.
// It must be injected as a pointer value.
//
// A system can have multiple independent Local parameters even with the same type T.
type Local[T any] struct {
	_     NoCopy
	Value T
}

func (l *Local[T]) newState(*World, localT) SystemParamState {
	return &localState{
		Type:  reflect.TypeFor[*Local[T]](),
		Value: reflect.ValueOf(l),
	}
}

type localT interface {
	newState(*World, localT) SystemParamState
}

type localState struct {
	Type  reflect.Type
	Value reflect.Value
}

func (l *localState) GetValue(SystemContext) (reflect.Value, error) {
	return l.Value, nil
}

func (l *localState) CleanupValue() {}

func (l *localState) ValueType() reflect.Type {
	return l.Type
}
