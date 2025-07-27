package bykebiten

func PointerTo[T any](value T) *T {
	return &value
}
