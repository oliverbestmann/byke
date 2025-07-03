package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/query"
	"iter"
	"reflect"
)

type Query[T any] struct {
	inner *innerQuery
	items iter.Seq[T]
}

func (q *Query[T]) set(inner *innerQuery) {
	q.inner = inner

	var target T
	var targetIf any = &target

	q.items = func(yield func(T) bool) {
		for ref := range q.inner.Iter {
			query.FromEntity(targetIf, q.inner.Setters, ref)

			if !yield(target) {
				return
			}
		}
	}
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
	for range q.inner.Iter {
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
	query.ParsedQuery
	Storage *arch.Storage
	Iter    iter.Seq[arch.EntityRef]
}

type queryAccessor interface {
	parse() (query.ParsedQuery, error)
	set(inner *innerQuery)
}
