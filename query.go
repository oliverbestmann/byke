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

func (*Query[T]) init(world *World) SystemParamState {
	var q Query[T]

	parsed, err := q.parse()
	if err != nil {
		queryType := reflect.TypeOf(q).Elem()
		panic(fmt.Sprintf("failed to parse query of type %s: %s", queryType, err))
	}

	inner := &innerQuery{
		Query:   parsed.Query,
		Setters: parsed.Setters,
		Storage: world.storage,
	}

	inner.iter = world.storage.IterQuery(&inner.Query)

	q.inner = inner
	q.items = makeQueryIter[T](inner)

	return &queryParamState{
		ptrToValue: reflect.ValueOf(&q),
		world:      world,
		mutable:    parsed.Mutable,
		inner:      inner,
	}
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

type queryParamState struct {
	ptrToValue reflect.Value

	world   *World
	inner   *innerQuery
	mutable []*arch.ComponentType
}

func (q *queryParamState) getValue(system *preparedSystem) reflect.Value {
	q.inner.Query.LastRun = system.LastRun
	return q.ptrToValue.Elem()
}

func (q *queryParamState) cleanupValue(value reflect.Value) {
	q.world.recheckComponents(q.mutable)
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
