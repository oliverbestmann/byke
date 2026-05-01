package byke

import (
	"reflect"
)

type Single[T any] struct {
	Value T
}

func (s Single[T]) NewState(world *World) SystemParamState {
	var query Query[T]
	queryState := query.NewState(world)

	var value Single[T]

	return &singleParamState{
		QueryState: queryState,
		Type:       reflect.TypeFor[Single[T]](),
		extractValue: func(q reflect.Value) (reflect.Value, error) {
			query := q.Addr().Interface().(*Query[T])

			singleValue, ok := query.Single()
			if !ok {
				return reflect.Value{}, ErrSkipSystem
			}

			value.Value = singleValue

			return reflect.ValueOf(&value).Elem(), nil
		},
	}
}

type singleParamState struct {
	QueryState   SystemParamState
	Type         reflect.Type
	extractValue func(q reflect.Value) (reflect.Value, error)
}

func (s *singleParamState) GetValue(sc SystemContext) (reflect.Value, error) {
	value, err := s.QueryState.GetValue(sc)
	if err != nil {
		return reflect.Value{}, err
	}

	return s.extractValue(value)
}

func (s *singleParamState) CleanupValue() {
	s.QueryState.CleanupValue()
}

func (s *singleParamState) ValueType() reflect.Type {
	return s.Type
}
