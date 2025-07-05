package byke

import (
	"reflect"
)

type StateType[S comparable] struct {
	InitialValue S
}

func (r StateType[S]) configureStateIn(app *App) {
	ValidateComponent[DespawnOnExitStateComponent[S]]()

	app.InsertResource(State[S]{current: r.InitialValue})
	app.InsertResource(NextState[S]{})

	app.AddSystems(StateTransition, performStateTransition[S])
	app.AddSystems(OnChange[S](), despawnOnExitStateSystem[S])
}

func DespawnOnExitState[S comparable](state S) DespawnOnExitStateComponent[S] {
	return DespawnOnExitStateComponent[S]{state: state}
}

type stateChangedScheduleId[S comparable] struct {
	stateType reflect.Type
	value     S

	enter  bool
	exit   bool
	change bool
}

func OnEnter[S comparable](stateValue S) ScheduleId {
	return stateChangedScheduleId[S]{
		stateType: reflect.TypeFor[S](),
		value:     stateValue,
		enter:     true,
	}
}

func OnExit[S comparable](stateValue S) ScheduleId {
	return stateChangedScheduleId[S]{
		stateType: reflect.TypeFor[S](),
		value:     stateValue,
		exit:      true,
	}
}

func OnChange[S comparable]() ScheduleId {
	return stateChangedScheduleId[S]{
		stateType: reflect.TypeFor[S](),
		change:    true,
	}
}

type State[S comparable] struct {
	current     S
	initialized bool
}

func (s State[S]) Current() S {
	return s.current
}

type NextState[S comparable] struct {
	isSet bool
	next  S
}

func (n *NextState[S]) Set(nextState S) {
	n.isSet = true
	n.next = nextState
}

func (n *NextState[S]) Clear() {
	var zeroState S

	n.isSet = false
	n.next = zeroState
}

type DespawnOnExitStateComponent[S comparable] struct {
	Component[DespawnOnExitStateComponent[S]]
	state S
}

func performStateTransition[S comparable](world *World, state *State[S], nextState *NextState[S]) {
	if !state.initialized {
		// we need to run the OnEnter schedule once
		state.initialized = true
		world.RunSchedule(OnEnter(state.current))
		return
	}

	if !nextState.isSet {
		return
	}

	if nextState.next == state.current {
		return
	}

	// keep the previous state value so we can trigger OnExit
	previousState := state.current

	// update the state resources
	state.current = nextState.next
	nextState.Clear()

	// run the OnExit / OnEnter schedules
	world.RunSchedule(OnChange[S]())
	world.RunSchedule(OnExit(previousState))
	world.RunSchedule(OnEnter(state.current))
}

type DespawnStateScopedItem[S comparable] struct {
	EntityId    EntityId
	StateScoped DespawnOnExitStateComponent[S]
}

func despawnOnExitStateSystem[S comparable](
	commands *Commands,
	state State[S],
	query Query[DespawnStateScopedItem[S]],
) {
	// TODO have a StateTransitionEvent event and offer OnExit and OnEnter
	for item := range query.Items() {
		if item.StateScoped.state != state.Current() {
			commands.Entity(item.EntityId).Despawn()
		}
	}
}
