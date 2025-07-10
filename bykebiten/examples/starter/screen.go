package main

import (
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
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

func spawnTitleScreenSystem(commands *Commands, screenSize ScreenSize) {
	fmt.Println("Spawn title screen")

	commands.Spawn(
		DespawnOnExitState(ScreenTitle),

		Sprite{
			Image: EbitenPNG(),
		},

		// palce at the center of the screen
		NewTransform().WithTranslation(screenSize.Mul(0.5)),

		ColorTint{
			Color: color.RGBA(1.0, 1.0, 1.0, 0.2),
		},
	)
}

func spawnGameScreenSystem(commands *Commands) {
	fmt.Println("Spawn game screen")
}
