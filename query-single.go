package byke

import (
	"reflect"
)

type Single[T any] struct {
	Value T
}

func (s Single[T]) init(world *World) SystemParamState {
	var query Query[T]
	queryState := query.init(world)

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

func (s *singleParamState) getValue(sc systemContext) (reflect.Value, error) {
	value, err := s.QueryState.getValue(sc)
	if err != nil {
		return reflect.Value{}, err
	}

	return s.extractValue(value)
}

func (s *singleParamState) cleanupValue() {
	s.QueryState.cleanupValue()
}

func (s *singleParamState) valueType() reflect.Type {
	return s.Type
}
