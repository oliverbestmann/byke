package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/inner"
	"iter"
	"reflect"
)

// Query is a strongly typed query instance.
type Query[T any] struct {
	inner.Type[T]
	delegate *erasedQuery

	// scratch space holding one item C in the query
	item T
}

func (q *Query[T]) Get() (value *T, ok bool) {
	for value := range q.Items() {
		return &value, true
	}

	return nil, false
}

func (q *Query[T]) MustGet() T {
	value, ok := q.Get()
	if !ok {
		panic(fmt.Sprintf("no value in query for %T", value))
	}

	return *value
}

func (q *Query[T]) Count() int {
	var count int

	for range q.Items() {
		count += 1
	}

	return count
}

func (q *Query[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		target := reflect.ValueOf(&q.item).Elem()

		delegate := q.delegate
		hasValue := delegate.parsed.hasValue
		putValue := delegate.parsed.putValue

		for _, entity := range delegate.world.entities {
			// quick check if the entity has matches the Query predicate
			if hasValue != nil && !hasValue(delegate.world, delegate.system, entity) {
				continue
			}

			if putValue(entity, target) {
				// success, the entity matches and we've filled the target
				if !yield(q.item) {
					return
				}
			}
		}
	}
}

func (*Query[T]) isQuery(queryAccessor) {}

func (q *Query[T]) set(inner *erasedQuery) {
	q.delegate = inner
}

func (q *Query[T]) get() *erasedQuery {
	return q.delegate
}
