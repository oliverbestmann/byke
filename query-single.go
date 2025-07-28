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
		extractValue: func(q reflect.Value) reflect.Value {
			query := q.Addr().Interface().(*Query[T])

			singleValue, ok := query.Single()
			if !ok {
				panic("query did not return a single result")
			}

			value.Value = singleValue

			return reflect.ValueOf(&value).Elem()
		},
	}
}

type singleParamState struct {
	QueryState   SystemParamState
	Type         reflect.Type
	extractValue func(q reflect.Value) reflect.Value
}

func (s *singleParamState) getValue(sc systemContext) reflect.Value {
	return s.extractValue(s.QueryState.getValue(sc))
}

func (s *singleParamState) cleanupValue() {
	s.QueryState.cleanupValue()
}

func (s *singleParamState) valueType() reflect.Type {
	return s.Type
}
