package main

import (
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
		InitialValue: MenuStateNone,
	})

	app.AddSystems(OnEnter(MenuStatePause), spawnPausedMenuSystem)
	app.AddSystems(OnEnter(MenuStateTitle), spawnTitleMenuSystem)

	app.AddSystems(Update, applyColorByInteractionState)
}

func button(pos gm.Vec, text string) ErasedComponent {
	// create a rectangle
	var buttonShape Path
	buttonShape.Rectangle(gm.RectWithCenterAndSize(gm.VecZero, gm.Vec{X: 192.0, Y: 48.0}))

	return BundleOf(
		UiCamera,

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
			UiCamera,
			Text{Text: text},
			Layer{Z: 1},
		),
	)
}

func spawnTitleMenuSystem(commands *Commands) {
	commands.
		Spawn(
			DespawnOnExitState(ScreenTitle),
			button(gm.Vec{Y: -32}, "Start game"),
		).
		Observe(func(_ On[Clicked], screenState *NextState[Screen], menuState *NextState[MenuState]) {
			screenState.Set(ScreenGame)
			menuState.Set(MenuStateNone)
		})

	commands.
		Spawn(
			DespawnOnExitState(ScreenTitle),
			button(gm.Vec{Y: 32}, "Exit Game"),
		).
		Observe(func(_ On[Clicked], exit *EventWriter[AppExit]) {
			exit.Write(AppExitSuccess)
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
	commands.
		Spawn(
			DespawnOnExitState(MenuStatePause),
			button(gm.Vec{Y: -32}, "Continue game"),
		).
		Observe(func(_ On[Clicked], pauseState *NextState[PauseState], menuState *NextState[MenuState]) {
			pauseState.Set(PauseStateUnpaused)
			menuState.Set(MenuStateNone)
		})

	commands.
		Spawn(
			DespawnOnExitState(MenuStatePause),
			button(gm.Vec{Y: 32}, "Back to menu"),
		).
		Observe(func(_ On[Clicked], nextScreen *NextState[Screen], pauseState *NextState[PauseState]) {
			nextScreen.Set(ScreenTitle)
			pauseState.Set(PauseStateUnpaused)
		})

}
