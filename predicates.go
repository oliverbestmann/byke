package byke

// InState returns a system predicate that prevents a system from running unless the given
// state value matches the expected state. Useful for conditional logic based on game states
// like menus, gameplay, paused, etc.
//
//	app.AddSystems(Update, System(doSomething).RunIf(InState(MenuStateTitle)))
func InState[S comparable](expectedState S) AnySystem {
	return System(func(state State[S]) bool {
		return state.Current() == expectedState
	})
}

// NotInState returns a system predicate that prevents a system from running when the given
// state value matches. Useful for excluding systems from running in specific states.
//
//	app.AddSystems(Update, System(doSomething).RunIf(NotInState(MenuStateTitle)))
func NotInState[S comparable](stateValue S) AnySystem {
	return System(func(state State[S]) bool {
		return state.Current() != stateValue
	})
}

// TimerJustFinished returns a system predicate that runs a system only in the frame when
// the provided timer finishes. Useful for one-shot actions like effects or state transitions.
func TimerJustFinished(timer Timer) AnySystem {
	return System(func(vt VirtualTime) bool {
		return timer.Tick(vt.Delta).JustFinished()
	})
}

// ResourceExists returns true if the given resource exists in the world.
// Use it with RunIf to conditionally run a system based on resource presence.
//
//	app.AddSystems(Update, System(doSomething).RunIf(ResourceExists[MyResource]))
func ResourceExists[T any](res ResOption[T]) bool {
	return res.Value != nil
}

// ResourceMissing returns true if the given resource does not exist in the world.
// Use it with RunIf to conditionally run a system based on resource absence.
//
//	app.AddSystems(Update, System(doSomething).RunIf(ResourceMissing[MyResource]))
func ResourceMissing[T any](res ResOption[T]) bool {
	return res.Value == nil
}
