package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
)

type PauseState int

const (
	PauseStatePaused   PauseState = 0
	PauseStateUnpaused PauseState = 1
)

func pluginPause(app *App) {
	app.InitState(StateType[PauseState]{
		InitialValue: PauseStateUnpaused,
	})

	app.AddSystems(PreUpdate, System(pauseGameSystem).
		RunIf(InState(PauseStateUnpaused)).
		RunIf(InState(ScreenGame)))

	app.AddSystems(OnExit(PauseStatePaused), unpauseGameSystem)
}

func pauseGameSystem(vt *VirtualTime, pauseState *NextState[PauseState], menuState *NextState[MenuState], keys bykebiten.Keys) {
	if keys.IsJustPressed(ebiten.KeyEscape) {
		pauseState.Set(PauseStatePaused)
		menuState.Set(MenuStatePause)
		vt.Scale = 0.0
	}
}

func unpauseGameSystem(vt *VirtualTime) {
	vt.Scale = 1.0
}
