package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/refl"
	"github.com/oliverbestmann/byke/internal/typedpool"
	"reflect"
)

var valueSlices = typedpool.New[[]reflect.Value]()

func prepareSystem(w *World, config SystemConfig) *preparedSystem {
	rSystem := config.fn

	if rSystem.Kind() != reflect.Func {
		panic(fmt.Sprintf("not a function: %s", rSystem.Type()))
	}

	preparedSystem := &preparedSystem{SystemConfig: config}

	systemType := rSystem.Type()

	// collect a number of functions that when called will prepare the systems parameters
	var params []SystemParamState

	for idx := range systemType.NumIn() {
		inType := systemType.In(idx)

		resourceCopy, resourceCopyOk := w.resources[reflect.PointerTo(inType)]
		resource, resourceOk := w.resources[inType]

		switch {
		case refl.ImplementsInterfaceDirectly[SystemParam](inType):
			params = append(params, makeSystemParamState(w, inType))

		case refl.ImplementsInterfaceDirectly[SystemParam](reflect.PointerTo(inType)):
			params = append(params, makeSystemParamState(w, inType))

		case inType == reflect.TypeFor[*World]():
			params = append(params, valueSystemParamState(reflect.ValueOf(w)))

		case resourceCopyOk:
			params = append(params, valueSystemParamState(resourceCopy.Reflect.Elem()))

		case resourceOk:
			params = append(params, valueSystemParamState(resource.Reflect.Value))

		default:
			panic(fmt.Sprintf("Can not handle system param of type %s", inType))
		}
	}

	// verify that all the param types match their actual types
	for idx, param := range params {
		inType := systemType.In(idx)
		if !param.valueType().AssignableTo(inType) {
			panic(fmt.Sprintf("Argument %d of %s is not assignable to param value of type %s", idx, systemType.Name(), inType))
		}
	}

	preparedSystem.RawSystem = func() {
		paramValues := valueSlices.Get()
		defer valueSlices.Put(paramValues)

		*paramValues = (*paramValues)[:0]

		for _, param := range params {
			*paramValues = append(*paramValues, param.getValue(preparedSystem))
		}

		rSystem.Call(*paramValues)

		for idx, param := range params {
			param.cleanupValue((*paramValues)[idx])
		}

		// clear any pointers that are still in the param slice
		clear(*paramValues)
	}

	return preparedSystem
}

func makeSystemParamState(world *World, ty reflect.Type) SystemParamState {
	for ty.Kind() == reflect.Pointer {
		ty = ty.Elem()
	}

	// allocate a new instance on the heap and get the value as an interface
	param := reflect.New(ty).Interface().(SystemParam)

	// initialize using the world
	return param.init(world)
}
