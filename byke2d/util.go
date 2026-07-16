package byke2d

func derefOr[T any](ptrToValue *T, fallbackValue T) T {
	if ptrToValue != nil {
		return *ptrToValue
	}

	return fallbackValue
}
