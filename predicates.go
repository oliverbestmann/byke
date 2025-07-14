package byke

func InState[S comparable](expectedState S) Systems {
	return System(func(state State[S]) bool {
		return state.Current() == expectedState
	})
}

func ResourceExists[T any](res ResOption[T]) bool {
	return res.Value != nil
}
