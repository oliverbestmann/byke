package byke

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/query"
	spoke "github.com/oliverbestmann/byke/spoke"
	"iter"
	"reflect"
)

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
		Query:   world.storage.OptimizeQuery(parsed.Builder.Build()),
		Setters: parsed.Setters,
		Storage: world.storage,
	}

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
	q.inner = inner
	q.items = makeQueryIter[T](inner)
}

func (q *Query[T]) parse() (query.ParsedQuery, error) {
	return query.ParseQuery(reflect.TypeFor[T]())
}

func (q *Query[T]) Get(entityId EntityId) (T, bool) {
	var target T

	ref, ok := q.inner.Storage.GetWithQuery(q.inner.Query, q.inner.QueryContext, entityId)
	if !ok {
		return target, false
	}

	query.FromEntity(&target, q.inner.Setters, ref)

	return target, true
}

func (q *Query[T]) Count() int {
	it := q.inner.Storage.IterQuery(q.inner.Query, q.inner.QueryContext)

	var count int
	for {
		_, more := it.Next()
		if !more {
			return count
		}

		count += 1
	}
}

func (q *Query[T]) Items() iter.Seq[T] {
	return q.items
}

func (q *Query[T]) AppendTo(values []T) []T {
	iterValues(q.inner, func(value T) bool {
		values = append(values, value)
		return true
	})

	return values
}

func (q *Query[T]) Single() (T, bool) {
	var result T
	var count int

	for value := range q.items {
		count += 1

		switch count {
		case 1:
			result = value

		case 2:
			break
		}
	}

	return result, count == 1
}

func (q *Query[T]) MustFirst() T {
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
	mutable []*spoke.ComponentType
}

func (q *queryParamState) getValue(sc systemContext) reflect.Value {
	q.inner.QueryContext.LastRun = sc.LastRun
	return q.ptrToValue.Elem()
}

func (q *queryParamState) cleanupValue() {
	q.world.recheckComponents(q.inner.Query, q.mutable)
}

func (q *queryParamState) valueType() reflect.Type {
	return q.ptrToValue.Type().Elem()
}

type innerQuery struct {
	Setters      []query.Setter
	Query        *spoke.CachedQuery
	Storage      *spoke.Storage
	QueryContext spoke.QueryContext
}

func iterValues[T any](inner *innerQuery, fn func(value T) bool) {
	it := inner.Storage.IterQuery(inner.Query, inner.QueryContext)

	var target T

	for {
		ref, more := it.Next()
		if !more {
			break
		}

		query.FromEntity(&target, inner.Setters, ref)

		if !fn(target) {
			break
		}
	}
}

func makeQueryIter[T any](inner *innerQuery) func(yield func(T) bool) {
	return func(yield func(T) bool) {
		iterValues[T](inner, yield)
	}
}
