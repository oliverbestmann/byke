package byke

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"strings"

	"github.com/oliverbestmann/byke/internal/refl"
	"github.com/oliverbestmann/byke/internal/typedpool"
	"github.com/oliverbestmann/byke/spoke"
)

var valueSlices = typedpool.New[[]reflect.Value]()

type systemTrigger struct {
	EventValue Event
}

type systemContext struct {
	// a value that has triggerd the execution of the system.
	// Should be an event.
	Trigger systemTrigger

	// last tick the system ran
	LastRun spoke.Tick
	InValue any
}

type preparedSystem struct {
	systemConfig

	Name        string
	LastRun     spoke.Tick
	RawSystem   func(systemContext) any
	IsPredicate bool

	// This system has a Commands parameter. We need this to handle flush point
	// generation between systems in the future
	HasCommands bool

	Predicates []*preparedSystem
}

func (w *World) prepareSystemUncached(config systemConfig) *preparedSystem {
	rSystem := config.SystemFunc

	if rSystem.Kind() != reflect.Func {
		panic(fmt.Sprintf("not a function: %s", rSystem.Type()))
	}

	preparedSystem := &preparedSystem{
		systemConfig: config,
		Name:         funcNameOf(config.SystemFunc),
	}

	slog.Info("Prepare system", slog.String("name", preparedSystem.Name), slog.Int("idx", len(w.systems)))

	systemType := rSystem.Type()

	// collect a number of functions that when called will prepare the systems parameters
	var params []SystemParamState

	for idx := range systemType.NumIn() {
		inType := systemType.In(idx)

		switch {
		case refl.ImplementsInterfaceDirectly[SystemParam](inType):
			params = append(params, makeSystemParamState(w, inType))

		case refl.ImplementsInterfaceDirectly[SystemParam](reflect.PointerTo(inType)):
			params = append(params, makeSystemParamState(w, inType))

		case inType == reflect.TypeFor[*World]():
			params = append(params, valueSystemParamState(reflect.ValueOf(w)))

		default:
			// in all other cases, we assume that it might be a resource that we can
			// grab during runtime
			params = append(params, makeResourceSystemParamState(w, inType))
		}
	}

	for idx, param := range params {
		inType := systemType.In(idx)

		// verify that all the param types match their actual types
		if !param.valueType().AssignableTo(inType) {
			panic(fmt.Sprintf(
				"Argument %d (%s) of %q is not assignable to value of type %s",
				idx, param.valueType(), preparedSystem.Name, inType,
			))
		}

		if inType == reflect.TypeFor[*Commands]() {
			preparedSystem.HasCommands = true
		}
	}

	// check the return values. we currently only allow a `bool` return value
	if systemType.NumOut() > 0 {
		if systemType.NumOut() > 1 {
			panic("System must have at most one return value")
		}

		returnType := systemType.Out(0)
		if returnType != reflect.TypeFor[bool]() {
			panic("for now, only bool is accepted as a return type of a system")
		}

		preparedSystem.IsPredicate = true
	}

	preparedSystem.RawSystem = func(sc systemContext) any {
		paramValues := valueSlices.Get()
		defer valueSlices.Put(paramValues)

		*paramValues = (*paramValues)[:0]

		sc.LastRun = preparedSystem.LastRun

		for idx, param := range params {
			value, err := param.getValue(sc)

			if err != nil {
				// need to cleanup the ones we've already added
				for _, param := range params[:idx+1] {
					param.cleanupValue()
				}

				if errors.Is(err, ErrSkipSystem) {
					return nil
				}

				panic(err)
			}

			*paramValues = append(*paramValues, value)
		}

		returnValues := rSystem.Call(*paramValues)

		for _, param := range params {
			param.cleanupValue()
		}

		// clear any pointers that are still in the param slice
		clear(*paramValues)

		// convert return value to interface
		var returnValue any
		if len(returnValues) == 1 {
			returnValue = returnValues[0].Interface()
		}

		return returnValue
	}

	// prepare predicate systems if any
	for _, predicate := range config.Predicates {
		for _, system := range asSystemConfigs(predicate) {
			predicateSystem := w.prepareSystem(system)
			if !predicateSystem.IsPredicate {
				panic("predicate system is not actually a predicate")
			}

			preparedSystem.Predicates = append(preparedSystem.Predicates, predicateSystem)
		}
	}

	return preparedSystem
}

func funcNameOf(fn reflect.Value) string {
	if fn.Kind() != reflect.Func {
		panic("not a function")
	}

	ptrToCode := uintptr(fn.UnsafePointer())
	funcValue := runtime.FuncForPC(ptrToCode)
	if funcValue == nil {
		return fmt.Sprintf("unknown(%s)", fn.Type())
	}

	name := funcValue.Name()
	if idx := strings.LastIndexByte(name, '/'); idx >= 0 {
		name = name[idx+1:]
	}

	return name
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
