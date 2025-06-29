package ecs

import (
	"reflect"
)

var StateTransition = &Schedule{}

type RegisterState[S comparable] struct {
	InitialValue S
}

func (r RegisterState[S]) configureStateIn(app *App) {
	app.InsertResource(State[S]{current: r.InitialValue})
	app.InsertResource(NextState[S]{})
	app.AddSystems(StateTransition, performStateTransition[S])
}

type stateChangedScheduleId[S comparable] struct {
	stateType reflect.Type
	value     S
	enter     bool
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
		enter:     false,
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

	world.RunSchedule(OnExit(state.current))

	state.current = nextState.next
	nextState.Clear()

	world.RunSchedule(OnEnter(state.current))
}
