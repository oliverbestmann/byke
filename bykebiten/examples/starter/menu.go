package main

import (
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
)

type MenuState int

const (
	MenuStateNone  MenuState = 0
	MenuStateTitle MenuState = 1
	MenuStatePause MenuState = 2
)

func pluginMenu(app *App) {
	app.InitState(StateType[MenuState]{
		InitialValue: MenuStateTitle,
	})

	app.AddSystems(OnEnter(Paused), spawnPausedMenuSystem)
	app.AddSystems(OnEnter(ScreenTitle), spawnTitleMenuSystem)
}

func spawnTitleMenuSystem(commands *Commands, screenSize ScreenSize) {
	commands.
		Spawn(
			DespawnOnExitState(ScreenTitle),
			NewTransform().WithTranslation(screenSize.Mul(0.5)),
			Text{Text: "Start game"},
			Clickable{},
		).
		Observe(func(_ On[Clicked], screenState *NextState[Screen]) {
			screenState.Set(ScreenGame)
		})
}

func spawnPausedMenuSystem(commands *Commands) {
	fmt.Println("Spawn paused menu")
}
