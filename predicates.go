package byke

// InState is a system predicate that can be passed to Systems.RunIf to only run a system
// if a given state has the expected value.
//
//	app.AddSystems(Update, System(doSomething).RunIf(InState(MenuStateTitle)))
func InState[S comparable](expectedState S) AnySystem {
	return System(func(state State[S]) bool {
		return state.Current() == expectedState
	})
}

// NotInState is a system predicate that can be passed to Systems.RunIf to only run a system
// if a given state does not have a specific value.
//
//	app.AddSystems(Update, System(doSomething).RunIf(NotInState(MenuStateTitle)))
func NotInState[S comparable](stateValue S) AnySystem {
	return System(func(state State[S]) bool {
		return state.Current() != stateValue
	})
}

// TimerJustFinished is a system predicate that can be passed to Systems.RunIf to only run a system
// if the provided timer has just finished.
func TimerJustFinished(timer Timer) AnySystem {
	return System(func(vt VirtualTime) bool {
		return timer.Tick(vt.Delta).JustFinished()
	})
}

// ResourceExists is a system predicate that can be passed to Systems.RunIf to only run a system
// if a given resource does exist in the world. Use it as a reference:
//
//	app.AddSystems(Update, System(doSomething).RunIf(ResourceExists[MyResource]))
func ResourceExists[T any](res ResOption[T]) bool {
	return res.Value != nil
}

// ResourceMissing is a system predicate thtat can be passed to Systems.RunIf to only run a system
// if a given resource does not exist in the world. Use it as a reference:
//
//	app.AddSystems(Update, System(doSomething).RunIf(ResourceMissing[MyResource]))
func ResourceMissing[T any](res ResOption[T]) bool {
	return res.Value == nil
}
