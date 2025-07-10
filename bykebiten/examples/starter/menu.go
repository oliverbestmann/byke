package main

import (
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
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

	// create a rectangle
	var buttonShape Path
	buttonShape.Rectangle(gm.RectWithCenterAndSize(gm.VecZero, gm.Vec{X: 128.0, Y: 48.0}))

	commands.
		Spawn(
			DespawnOnExitState(ScreenTitle),
			NewTransform().WithTranslation(screenSize.Mul(0.5)),

			buttonShape,
			Fill{
				Color: color.RGBA(0.4, 0.2, 0.4, 1.0),
			},
			Stroke{
				Width: 2.0,
				Color: color.RGBA(0.2, 0.0, 0.2, 1.0),
			},

			Interactable{},

			SpawnChild(
				Text{Text: "Start game"},
				Layer{Z: 1},
			),
		).
		Observe(func(_ On[Clicked], screenState *NextState[Screen]) {
			screenState.Set(ScreenGame)
		}).
		Observe(func(trigger On[PointerOver], query Query[*Fill]) {
			fill, _ := query.Get(trigger.Target)
			fill.Color = color.RGBA(0.8, 0.2, 0.6, 1.0)
		}).
		Observe(func(trigger On[PointerOut], query Query[*Fill]) {
			fill, _ := query.Get(trigger.Target)
			fill.Color = color.RGBA(0.4, 0.2, 0.4, 1.0)
		})

}

func spawnPausedMenuSystem(commands *Commands) {
	fmt.Println("Spawn paused menu")
}
