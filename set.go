package ecs

import (
	"iter"
	"maps"
)

type Set[T comparable] struct {
	values map[T]struct{}
}

func (s *Set[T]) Insert(value T) {
	if s.values == nil {
		s.values = make(map[T]struct{})
	}

	s.values[value] = struct{}{}
}

func (s *Set[T]) Remove(value T) {
	delete(s.values, value)
}

func (s *Set[T]) Has(value T) bool {
	_, ok := s.values[value]
	return ok
}

func (s *Set[T]) Iter() iter.Seq[T] {
	return maps.Keys(s.values)
}

func (s *Set[T]) PopOne() (T, bool) {
	for value := range s.values {
		return value, true
	}

	var tNil T
	return tNil, false
}

func (s *Set[T]) Len() int {
	return len(s.values)
}
