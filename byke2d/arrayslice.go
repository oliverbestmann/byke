package byke2d

type ArraySlice[T any] struct {
	values [16]T
	len    int
}

func ArraySliceOf[T any](values []T) ArraySlice[T] {
	if len(values) > 16 {
		panic("too many values")
	}

	var res ArraySlice[T]
	res.len = len(values)
	copy(res.values[:], values)
	return res
}

func (a *ArraySlice[T]) AsSlice() []T {
	return a.values[:a.len]
}

func (a *ArraySlice[T]) Len() int {
	return a.len
}

func (a *ArraySlice[T]) Append(value T) {
	if a.len == 16 {
		panic("too many values")
	}

	a.values[a.len] = value
	a.len += 1
}
