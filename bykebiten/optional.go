package bykebiten

type Optional[T any] struct {
	Value T
	IsSet bool
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{
		Value: value,
		IsSet: true,
	}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func (o *Optional[T]) Or(fallbackValue T) T {
	if !o.IsSet {
		return fallbackValue
	}

	return o.Value
}
