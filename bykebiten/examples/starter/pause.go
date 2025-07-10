package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
)

type PauseState int

const (
	Paused   PauseState = 0
	Unpaused PauseState = 1
)

func pluginPause(app *App) {
	app.InitState(StateType[PauseState]{
		InitialValue: Unpaused,
	})

	app.AddSystems(PreUpdate, System(pauseGameSystem).RunIf(InState(Unpaused)).RunIf(InState(ScreenGame)))
	app.AddSystems(OnExit(Paused), unpauseGameSystem)
}

func pauseGameSystem(vt *VirtualTime, pauseState *NextState[PauseState], keys bykebiten.Keys) {
	if keys.IsJustPressed(ebiten.KeyEscape) {
		pauseState.Set(Paused)
		vt.Scale = 0.0
	}
}

func unpauseGameSystem(vt *VirtualTime) {
	vt.Scale = 1.0
}
