package byke

import (
	"errors"
	"reflect"
)

var ErrSkipSystem = errors.New("skip system")

// MakeSystemParam is a way to give a type special behaviour when it is used
// as a parameter to a system.
//
// While a system is being prepared, byke will check each parameter if it can be
// created using a registered MakeSystemParam function.
// If a parameter type can be created, a SystemParamState is cached.
//
// If a type is not supported, MakeSystemParam returns nil.
type MakeSystemParam func(world *World, pType reflect.Type) SystemParamState

// SystemParamState is the state produced by SystemParam.
// TODO i need to check this interface & the assumptions,
//
//	maybe it makes sense to merge it into SystemParam.
type SystemParamState interface {
	// GetValue returns the value that should be passed to the system.
	// While this might be the same value as the SystemParam that has created the SystemParamState,
	// it must be of the same type as the SystemParam.
	//
	// In case that the parameter has no value, a zero reflect.Value is to be returned.
	GetValue(sc SystemContext) (reflect.Value, error)

	// CleanupValue will be called once the system is executed. It is used
	// to e.g. apply a Commands object against the world
	CleanupValue()

	// ValueType returns the exact type that getValue will return. This is used
	// while preparing
	ValueType() reflect.Type
}

// valueSystemParamState is a simple implementation of SystemParamState
// that just returns a constant value
type valueSystemParamState reflect.Value

func (s valueSystemParamState) GetValue(SystemContext) (reflect.Value, error) {
	return reflect.Value(s), nil
}

func (s valueSystemParamState) ValueType() reflect.Type {
	return reflect.Value(s).Type()
}

func (valueSystemParamState) CleanupValue() {
	// do nothing
}
