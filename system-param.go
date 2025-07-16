package byke

import "reflect"

// SystemParam is an interface to give a type special behaviour when it is used
// as a parameter to a system.
//
// While a system is being prepared, byke will check each parameter if it fulfills
// the SystemParam interface. If a parameter type does, a new instance will be allocate
// and the init method will be called.
//
// See Local, ResOption or Query for some implementations of SystemParam.
type SystemParam interface {
	// Init will be called while the system is being prepared.
	// It should setup everything as needed, e.g. allocate memory
	init(world *World) SystemParamState
}

// SystemParamState is the state produced by SystemParam.
// TODO i need to check this interface & the assumptions,
//
//	maybe it makes sense to merge it into SystemParam.
type SystemParamState interface {
	// getValue returns the value that should be passed to the system.
	// While this might be the same value as the SystemParam that has created the SystemParamState,
	// it must be of the same type as the SystemParam.
	//
	// In case that the parameter has no value, a zero reflect.Value is to be returned.
	getValue(sc systemContext) reflect.Value

	// cleanupValue will be called once the system is executed. It is used
	// to e.g. apply a Commands object against the world
	cleanupValue()

	// valueType returns the exact type that getValue will return. This is used
	// while preparing
	valueType() reflect.Type
}

// valueSystemParamState is a simple implementation of SystemParamState
// that just returns a constant value
type valueSystemParamState reflect.Value

func (s valueSystemParamState) getValue(systemContext) reflect.Value {
	return reflect.Value(s)
}

func (s valueSystemParamState) valueType() reflect.Type {
	return reflect.Value(s).Type()
}

func (valueSystemParamState) cleanupValue() {
	// do nothing
}
