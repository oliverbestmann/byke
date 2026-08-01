package meh

import (
	"iter"
)

type Key[T Comparable[T]] interface {
	Comparable[T]
	Hash() uint32
}

type Comparable[T Comparable[T]] interface {
	// EqualTo returns true, if both values are supposed to be "equal"
	EqualTo(other T) bool
}

type item[K Comparable[K], V any] struct {
	Key   K
	Value V
}

// Map is a simple map type that uses linear search to find keys.
// It compares key values using the Comparable interface which can also
// be implemented for types that are not "golang comparable".
type Map[K Key[K], V any] struct {
	items map[uint32][]item[K, V]
}

func (m *Map[K, V]) Insert(key K, value V) bool {
	hash, idx := m.indexOf(key)

	if idx >= 0 {
		bucket := m.items[hash]
		bucket[idx].Key = key
		bucket[idx].Value = value
		return false
	}

	if m.items == nil {
		m.items = map[uint32][]item[K, V]{}
	}

	m.items[hash] = append(m.items[hash], item[K, V]{
		Key:   key,
		Value: value,
	})

	return true
}

func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	hash, idx := m.indexOf(key)
	if idx < 0 {
		return
	}

	value = m.items[hash][idx].Value
	return value, true
}

func (m *Map[K, V]) Remove(key K) (value V, ok bool) {
	hash, idx := m.indexOf(key)
	if idx < 0 {
		return
	}

	bucket := m.items[hash]
	value = bucket[idx].Value

	if idx == 0 {
		delete(m.items, hash)
		return value, true
	}

	lastIdx := len(bucket) - 1
	if idx != lastIdx {
		// if not last index, swap with last value
		bucket[idx] = bucket[lastIdx]
	}

	// clear the last value and shrink the slices
	bucket[lastIdx] = item[K, V]{}
	bucket = bucket[:lastIdx]

	m.items[hash] = bucket

	return value, true
}

func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for _, bucket := range m.items {
			for _, item := range bucket {
				if !yield(item.Key) {
					return
				}
			}
		}
	}
}

func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, bucket := range m.items {
			for _, item := range bucket {
				if !yield(item.Value) {
					return
				}
			}
		}
	}
}

func (m *Map[K, V]) Items() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, bucket := range m.items {
			for _, item := range bucket {
				if !yield(item.Key, item.Value) {
					return
				}
			}
		}
	}
}

func (m *Map[K, V]) indexOf(key K) (uint32, int) {
	hash := key.Hash()

	bucket := m.items[hash]
	for idx := range bucket {
		if key.EqualTo(bucket[idx].Key) {
			return hash, idx
		}
	}

	return hash, -1
}
