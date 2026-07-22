package byke2d

type Area[T any] struct {
	// TODO we could go with multiple smaller chunks and then
	//  trash them on tick if we have too many
	chunk []T
}

func (a *Area[T]) Tick() {
	clear(a.chunk)
	a.chunk = a.chunk[:0]
}

func (a *Area[T]) Alloc(m T) *T {
	if a.chunk == nil {
		a.chunk = make([]T, 0, 1024)
	}

	a.chunk = append(a.chunk, m)
	return &a.chunk[len(a.chunk)-1]
}
