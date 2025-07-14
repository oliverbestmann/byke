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

	app.AddSystems(Update, applyColorByInteractionState)
}

func button(pos gm.Vec, text string) ErasedComponent {
	// create a rectangle
	var buttonShape Path
	buttonShape.Rectangle(gm.RectWithCenterAndSize(gm.VecZero, gm.Vec{X: 128.0, Y: 48.0}))

	return Bundle(
		NewTransform().WithTranslation(pos),

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
			Text{Text: text},
			Layer{Z: 1},
		),
	)
}

func spawnTitleMenuSystem(commands *Commands, screenSize ScreenSize) {
	commands.
		Spawn(
			DespawnOnExitState(ScreenTitle),
			button(screenSize.Mul(0.5).Add(gm.Vec{Y: -32}), "Start game"),
		).
		Observe(func(_ On[Clicked], screenState *NextState[Screen], menuState *NextState[MenuState]) {
			screenState.Set(ScreenGame)
			menuState.Set(MenuStateNone)
		})
	commands.
		Spawn(
			DespawnOnExitState(ScreenTitle),
			button(screenSize.Mul(0.5).Add(gm.Vec{Y: 32}), "Settings"),
		).
		Observe(func(_ On[Clicked], screenState *NextState[Screen], menuState *NextState[MenuState]) {
			screenState.Set(ScreenGame)
			menuState.Set(MenuStateNone)
		})
}

type applyColorByInteractionStateQueryItem struct {
	Changed[InteractionState]
	InteractionState InteractionState
	Fill             *Fill
}

func applyColorByInteractionState(query Query[applyColorByInteractionStateQueryItem]) {
	for item := range query.Items() {
		if item.InteractionState == InteractionStateHover {
			item.Fill.Color = color.RGBA(0.8, 0.2, 0.6, 1.0)
		} else {
			item.Fill.Color = color.RGBA(0.4, 0.2, 0.4, 1.0)
		}
	}
}

func spawnPausedMenuSystem(commands *Commands) {
	fmt.Println("Spawn paused menu")
}
