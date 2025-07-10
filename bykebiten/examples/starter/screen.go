package main

import (
	"fmt"
	. "github.com/oliverbestmann/byke"
)

type Screen int

const (
	ScreenTitle Screen = 1
	ScreenGame  Screen = 2
)

func pluginScreen(app *App) {
	app.InitState(StateType[Screen]{
		InitialValue: ScreenTitle,
	})

	app.AddSystems(OnEnter(ScreenTitle), spawnTitleScreenSystem)
	app.AddSystems(OnEnter(ScreenGame), spawnGameScreenSystem)
}

func spawnTitleScreenSystem(commands *Commands) {
	fmt.Println("Spawn title screen")
}

func spawnGameScreenSystem(commands *Commands) {
	fmt.Println("Spawn game screen")
}
