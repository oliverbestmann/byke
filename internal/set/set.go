package set

import (
	"iter"
	"maps"
)

// Set provides a wrapper around a map[T]struct{}.
type Set[T comparable] struct {
	values map[T]struct{}
}

func FromValues[T comparable](values iter.Seq[T]) Set[T] {
	var result Set[T]

	result.values = make(map[T]struct{})

	for value := range values {
		result.values[value] = struct{}{}
	}

	return result
}

func (s *Set[T]) Insert(value T) bool {
	if s.values == nil {
		s.values = make(map[T]struct{})
	}

	// check if the value exists
	if _, exists := s.values[value]; exists {
		return false
	}

	// insert value
	s.values[value] = struct{}{}
	return true
}

func (s *Set[T]) Remove(value T) {
	delete(s.values, value)
}

func (s *Set[T]) Has(value T) bool {
	_, exists := s.values[value]
	return exists
}

func (s *Set[T]) Values() iter.Seq[T] {
	return maps.Keys(s.values)
}

func (s *Set[T]) Len() int {
	return len(s.values)
}

func (s *Set[T]) PopOne() (T, bool) {
	for value := range s.values {
		return value, true
	}

	var tNil T
	return tNil, false
}

func (s *Set[T]) InsertAll(values iter.Seq[T]) {
	for value := range values {
		s.Insert(value)
	}
}

func (s *Set[T]) Clone() Set[T] {
	return Set[T]{values: maps.Clone(s.values)}
}
