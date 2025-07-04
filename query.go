package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/query"
	"iter"
	"reflect"
)

type queryAccessor interface {
	parse() (query.ParsedQuery, error)
	set(inner *innerQuery)
}

type Query[T any] struct {
	inner *innerQuery
	items iter.Seq[T]
}

func (q *Query[T]) set(inner *innerQuery) {
	inner.iter = inner.Storage.IterQuery(&inner.Query)

	q.inner = inner
	q.items = makeQueryIter[T](inner)
}

func (q *Query[T]) parse() (query.ParsedQuery, error) {
	return query.ParseQuery(reflect.TypeFor[T]())
}

func (q *Query[T]) Get(entityId EntityId) (T, bool) {
	var target T

	ref, ok := q.inner.Storage.GetWithQuery(&q.inner.Query, entityId)
	if !ok {
		return target, false
	}

	query.FromEntity(&target, q.inner.Setters, ref)

	return target, true
}

func (q *Query[T]) Count() int {
	var count int
	for range q.inner.iter {
		count += 1
	}

	return count
}

func (q *Query[T]) Items() iter.Seq[T] {
	return q.items
}

func (q *Query[T]) MustGet() T {
	for value := range q.items {
		return value
	}

	var target T
	panic(fmt.Sprintf("no values in query for type %T", target))
}

type innerQuery struct {
	Setters []query.Setter
	Query   arch.Query
	Storage *arch.Storage
	iter    iter.Seq[arch.EntityRef]
}

func makeQueryIter[T any](inner *innerQuery) func(yield func(T) bool) {
	var target T
	var targetIf any = &target

	return func(yield func(T) bool) {
		for ref := range inner.iter {
			query.FromEntity(targetIf, inner.Setters, ref)

			if !yield(target) {
				return
			}
		}
	}
}
