package main

import (
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
}

func spawnTitleScreenSystem(commands *Commands,
	assets *Assets,
	menuState *NextState[MenuState]) {
	menuState.Set(MenuStateTitle)

	commands.Spawn(
		UiCamera,
		DespawnOnExitState(ScreenTitle),

		Sprite{
			Image: assets.Image("ebiten.png").Await(),
		},

		ColorTint{
			Color: color.RGBA(1.0, 1.0, 1.0, 0.2),
		},
	)
}
