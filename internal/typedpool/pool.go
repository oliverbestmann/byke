package typedpool

import "sync"

type Pool[T any] struct {
	pool sync.Pool
}

func New[T any]() *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any { return new(T) },
		},
	}
}

func (p *Pool[T]) Get() *T {
	cachedValue := p.pool.Get().(*T)
	return cachedValue
}

func (p *Pool[T]) Put(value *T) {
	p.pool.Put(value)
}
