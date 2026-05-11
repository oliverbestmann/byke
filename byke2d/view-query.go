package byke2d

import (
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/internal/refl"
)

// CurrentView marks the current view. A ViewQuery will limit its result
// to only the CurrentView.
type CurrentView byke.EntityId

type ViewQuery[T any] struct {
	value T
}

func (v ViewQuery[T]) Get() T {
	return v.value
}

func (v ViewQuery[T]) newState(world *byke.World, _ viewQueryT) byke.SystemParamState {
	type QueryValues[T any] struct {
		_ byke.With[Camera]

		ViewValue T
	}

	// instantiate a query that we can delegate to
	queryState := byke.NewQuerySystemParamState[QueryValues[T]](world)

	var viewQueryValue ViewQuery[T]

	return &viewQueryParamState{
		QueryState: queryState,
		Type:       reflect.TypeFor[ViewQuery[T]](),
		extractValue: func(q reflect.Value) (reflect.Value, error) {
			viewQuery := q.Addr().Interface().(*byke.Query[QueryValues[T]])

			currentView, ok := byke.ResourceOf[CurrentView](world)
			if !ok {
				// no current view, skipping this system
				return reflect.Value{}, byke.ErrSkipSystem
			}

			singleValue, ok := viewQuery.Get(byke.EntityId(*currentView))
			if !ok {
				return reflect.Value{}, byke.ErrSkipSystem
			}

			viewQueryValue.value = singleValue.ViewValue

			return reflect.ValueOf(&viewQueryValue).Elem(), nil
		},
	}
}

func makeViewQuery(world *byke.World, pType reflect.Type) byke.SystemParamState {
	if !refl.ImplementsInterfaceDirectly[viewQueryT](pType) {
		return nil
	}

	// the interface is not implemented on a pointer type,
	// we can create a pointer to it, and then dereference it to a viewQueryT
	t := reflect.New(pType).Elem().Interface().(viewQueryT)
	return t.newState(world, t)
}

type viewQueryT interface {
	newState(world *byke.World, _ viewQueryT) byke.SystemParamState
}

type viewQueryParamState struct {
	QueryState   byke.SystemParamState
	Type         reflect.Type
	extractValue func(q reflect.Value) (reflect.Value, error)
}

func (s *viewQueryParamState) GetValue(sc byke.SystemContext) (reflect.Value, error) {
	value, err := s.QueryState.GetValue(sc)
	if err != nil {
		return reflect.Value{}, err
	}

	return s.extractValue(value)
}

func (s *viewQueryParamState) CleanupValue() {
	s.QueryState.CleanupValue()
}

func (s *viewQueryParamState) ValueType() reflect.Type {
	return s.Type
}
