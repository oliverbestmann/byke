package byke

import (
	"fmt"
	"reflect"
)

type StateType[S comparable] struct {
	InitialValue S
}

func (r StateType[S]) configureStateIn(app *App) {
	ValidateComponent[despawnOnExitStateComponent[S]]()
	ValidateComponent[despawnOnEnterStateComponent[S]]()
	ValidateComponent[despawnOnStateTransitionComponent[S]]()

	app.InsertResource(State[S]{current: r.InitialValue})
	app.InsertResource(NextState[S]{})

	app.AddMessage(MessageType[StateTransitionEvent[S]]())

	app.AddSystems(StateTransition, System(
		performStateTransition[S],
		despawnOnExitStateSystem[S],
		despawnOnEnterStateSystem[S],
	).Chain())
}

type StateTransitionEvent[S comparable] struct {
	PreviousState S
	CurrentState  S
}

func (t *StateTransitionEvent[S]) IsIdentity() bool {
	return t.PreviousState == t.CurrentState
}

func DespawnOnExitState[S comparable](state S) despawnOnExitStateComponent[S] {
	return despawnOnExitStateComponent[S]{state: state}
}

func DespawnOnEnterState[S comparable](state S) despawnOnEnterStateComponent[S] {
	return despawnOnEnterStateComponent[S]{state: state}
}

type stateChangedScheduleId[S comparable] struct {
	stateType reflect.Type
	prevValue S
	currValue S

	enter      bool
	exit       bool
	transition bool
}

func (s stateChangedScheduleId[S]) String() string {
	switch {
	case s.enter:
		return fmt.Sprintf("OnEnter[%s](%v)", s.stateType, s.currValue)

	case s.exit:
		return fmt.Sprintf("OnExit[%s](%v)", s.stateType, s.prevValue)

	case s.transition:
		return fmt.Sprintf("OnTransition[%s](%v -> %v)", s.stateType, s.prevValue, s.currValue)

	default:
		panic("invalid stateChangedScheduleId")
	}
}

func (stateChangedScheduleId[S]) isSchedule() {}

func OnEnter[S comparable](stateValue S) ScheduleId {
	return stateChangedScheduleId[S]{
		stateType: reflect.TypeFor[S](),
		currValue: stateValue,
		enter:     true,
	}
}

func OnExit[S comparable](stateValue S) ScheduleId {
	return stateChangedScheduleId[S]{
		stateType: reflect.TypeFor[S](),
		prevValue: stateValue,
		exit:      true,
	}
}

func OnTransition[S comparable](previousStateValue, currentStateValue S) ScheduleId {
	return stateChangedScheduleId[S]{
		stateType:  reflect.TypeFor[S](),
		prevValue:  previousStateValue,
		currValue:  currentStateValue,
		transition: true,
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
	_     noCopy
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

type despawnOnExitStateComponent[S comparable] struct {
	Component[despawnOnExitStateComponent[S]]
	state S
}

type despawnOnEnterStateComponent[S comparable] struct {
	Component[despawnOnEnterStateComponent[S]]
	state S
}

type despawnOnStateTransitionComponent[S comparable] struct {
	Component[despawnOnStateTransitionComponent[S]]
	prevState S
	newState  S
}

func performStateTransition[S comparable](
	world *World,
	state *State[S],
	nextState *NextState[S],
	transitions *MessageWriter[StateTransitionEvent[S]],
) {
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

	// keep the previous state currValue so we can trigger OnExit
	previousState := state.current

	// update the state resources
	state.current = nextState.next
	nextState.Clear()

	// send transition event
	transitions.Write(StateTransitionEvent[S]{
		PreviousState: previousState,
		CurrentState:  state.current,
	})

	// run the OnExit / OnEnter schedules
	world.RunSchedule(OnExit(previousState))
	world.RunSchedule(OnTransition[S](previousState, state.current))
	world.RunSchedule(OnEnter(state.current))
}

type despawnOnExitStateScopedItem[S comparable] struct {
	EntityId      EntityId
	DespawnOnExit despawnOnExitStateComponent[S]
}

func despawnOnExitStateSystem[S comparable](
	commands *Commands,
	query Query[despawnOnExitStateScopedItem[S]],
	transitions *MessageReader[StateTransitionEvent[S]],
) {
	events := transitions.Read()
	if len(events) == 0 {
		return
	}

	// ignore identity transitions
	transition := events[len(events)-1]
	if transition.IsIdentity() {
		return
	}

	for item := range query.Items() {
		if item.DespawnOnExit.state == transition.PreviousState {
			commands.Entity(item.EntityId).Despawn()
		}
	}
}

type despawnOnEnterStateScopedItem[S comparable] struct {
	EntityId       EntityId
	DespawnOnEnter despawnOnEnterStateComponent[S]
}

func despawnOnEnterStateSystem[S comparable](
	commands *Commands,
	query Query[despawnOnEnterStateScopedItem[S]],
	transitions *MessageReader[StateTransitionEvent[S]],
) {
	events := transitions.Read()
	if len(events) == 0 {
		return
	}

	// ignore identity transitions
	transition := events[len(events)-1]
	if transition.IsIdentity() {
		return
	}

	for item := range query.Items() {
		if item.DespawnOnEnter.state == transition.CurrentState {
			commands.Entity(item.EntityId).Despawn()
		}
	}
}
