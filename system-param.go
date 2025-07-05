package byke

import "reflect"

type SystemParam interface {
	// Init will be called while the system is being prepared.
	// It should setup everything as needed
	init(world *World) SystemParamState
}

type SystemParamState interface {
	// Get returns the value that should be passed to the system.
	// This might be the same as SystemParam itself.
	//It should have the same type as SystemParam.
	getValue(system *preparedSystem) reflect.Value

	// Cleanup will be called once the system is executed. It is used
	// to e.g. apply a Commands object against the world
	cleanupValue(value reflect.Value)
}

type valueSystemParamState reflect.Value

func (s valueSystemParamState) getValue(*preparedSystem) reflect.Value {
	return reflect.Value(s)
}

func (valueSystemParamState) cleanupValue(reflect.Value) {
	// do nothing
}
