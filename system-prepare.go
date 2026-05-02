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
	"github.com/oliverbestmann/puffin-go"
)

var valueSlices = typedpool.New[[]reflect.Value]()

type systemTrigger struct {
	EventValue Event
}

type SystemContext struct {
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
	RawSystem   func(SystemContext) any
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

	defer puffin.NewScopeWithValue("byke.PrepareSystem", preparedSystem.Name).End()

	slog.Info("Prepare system", slog.String("name", preparedSystem.Name), slog.Int("idx", len(w.systems)))

	systemType := rSystem.Type()

	// collect a number of functions that when called will prepare the systems parameters
	var params []SystemParamState

	for idx := range systemType.NumIn() {
		inType := systemType.In(idx)

		if param, ok := w.makeSystemParams.newState(w, inType); ok {
			params = append(params, param)
			continue
		}

		// in all other cases, we assume that it might be a resource that we can
		// grab during runtime
		params = append(params, makeResourceSystemParamState(w, inType))
	}

	for idx, param := range params {
		inType := systemType.In(idx)

		// verify that all the param types match their actual types
		if !param.ValueType().AssignableTo(inType) {
			panic(fmt.Sprintf(
				"Argument %d (%s) of %q is not assignable to value of type %s",
				idx, param.ValueType(), preparedSystem.Name, inType,
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

	preparedSystem.RawSystem = func(sc SystemContext) any {
		paramValues := valueSlices.Get()
		defer valueSlices.Put(paramValues)

		*paramValues = (*paramValues)[:0]

		sc.LastRun = preparedSystem.LastRun

		for idx, param := range params {
			value, err := param.GetValue(sc)

			if err != nil {
				// need to cleanup the ones we've already added
				for _, param := range params[:idx+1] {
					param.CleanupValue()
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
			param.CleanupValue()
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

func makeWorldSystemParamState(world *World, pType reflect.Type) SystemParamState {
	if pType != reflect.TypeFor[*World]() {
		return nil
	}

	// return the world!
	return valueSystemParamState(reflect.ValueOf(world))
}

type makeSystemParams []MakeSystemParam

func (m makeSystemParams) newState(world *World, ty reflect.Type) (SystemParamState, bool) {
	for _, fn := range m {
		param := fn(world, ty)
		if param != nil {
			return param, true
		}
	}

	// no function supported creating system params of this type
	return nil, false
}

type genericNewState[T genericNewState[T]] interface {
	newState(*World, T) SystemParamState
}

func forwardToNewState[T genericNewState[T]](world *World, pType reflect.Type) SystemParamState {
	if !refl.ImplementsInterfaceDirectly[T](pType) {
		return nil
	}

	// if the interface is implemented on a pointer type *T, we need to do new(T)
	if pType.Kind() == reflect.Pointer {
		t := reflect.New(pType.Elem()).Interface().(T)
		return t.newState(world, t)
	}

	// if the interface is not implemented on a pointer type, we can create a pointer
	// to it, and then dereference it to a T
	t := reflect.New(pType).Elem().Interface().(T)
	return t.newState(world, t)
}

// Same as forwardToNewState with the difference, that the genericT is implemented on *pType, not on pType directly.
func forwardToNewStateOnPointer[T genericNewState[T]](world *World, pType reflect.Type) SystemParamState {
	return forwardToNewState[T](world, reflect.PointerTo(pType))
}
