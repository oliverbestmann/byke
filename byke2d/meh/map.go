package meh

import "iter"

type Comparable[T Comparable[T]] interface {
	// EqualTo returns true, if both values are supposed to be "equal"
	EqualTo(other T) bool
}

// Map is a simple map type that uses linear search to find keys.
// It compares key values using the Comparable interface which can also
// be implemented for types that are not "golang comparable".
type Map[K Comparable[K], V any] struct {
	keys   []K
	values []V
}

func (m *Map[K, V]) Insert(key K, value V) bool {
	idx := m.indexOf(key)
	if idx >= 0 {
		m.keys[idx] = key
		m.values[idx] = value
		return false
	}

	m.keys = append(m.keys, key)
	m.values = append(m.values, value)
	return true
}

func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	idx := m.indexOf(key)
	if idx < 0 {
		return
	}

	value = m.values[idx]
	return value, true
}

func (m *Map[K, V]) Remove(key K) (value V, ok bool) {
	idx := m.indexOf(key)
	if idx < 0 {
		return
	}

	value = m.values[idx]

	lastIdx := len(m.values) - 1
	if idx != lastIdx {
		// if not last index, swap with last value
		m.keys[idx] = m.keys[lastIdx]
		m.values[idx] = m.values[lastIdx]
	}

	// clear the last value and shring the slices
	var kZero K
	m.keys[lastIdx] = kZero
	m.keys = m.keys[:lastIdx]

	var vZero V
	m.values[lastIdx] = vZero
	m.values = m.values[:lastIdx]

	return value, true
}

func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for _, key := range m.keys {
			if !yield(key) {
				return
			}
		}
	}
}

func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, value := range m.values {
			if !yield(value) {
				return
			}
		}
	}
}

func (m *Map[K, V]) Items() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for idx, key := range m.keys {
			if !yield(key, m.values[idx]) {
				return
			}
		}
	}
}

func (m *Map[K, V]) indexOf(key K) int {
	for idx := range m.keys {
		if key.EqualTo(m.keys[idx]) {
			return idx
		}
	}

	return -1
}
