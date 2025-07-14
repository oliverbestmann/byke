package main

import (
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	. "github.com/oliverbestmann/byke/gm"
)

var _ = ValidateComponent[Ducky]()

func pluginGame(app *App) {
	app.AddSystems(OnEnter(ScreenGame), spawnGameScreenSystem)
	app.AddSystems(Update, updateAnimation)
}

type Ducky struct {
	Component[Ducky]
	AnimationTimer Timer
	AnimationIndex int
	Walking        bool
}

func spawnGameScreenSystem(commands *Commands) {
	fmt.Println("Spawn ducky")

	commands.Spawn(
		TransformFromXY(400, 300),
		Ducky{
			AnimationTimer: NewTimerFromSeconds(0.25, TimerModeRepeating),
		},
		Sprite{
			Image:      AssetDucky(),
			CustomSize: Some(VecSplat(96.0)),
		},
		Tiles{
			Rows:    2,
			Columns: 6,
			Width:   32,
			Height:  32,
			GapX:    1,
			GapY:    1,
		},
	)
}

func updateAnimation(vt VirtualTime, query Query[struct {
	Ducky     *Ducky
	TileIndex *TileIndex
	Sprite    *Sprite
}]) {
	item, ok := query.Single()
	if !ok {
		return
	}

	// update the frame counter
	item.Ducky.AnimationIndex += item.Ducky.AnimationTimer.Tick(vt.Delta).TimesFinishedThisTick()

	// update depending on animation state
	switch {
	case item.Ducky.Walking:
		item.TileIndex.Index = 6 + item.Ducky.AnimationIndex%6

	case !item.Ducky.Walking:
		item.TileIndex.Index = 0 + item.Ducky.AnimationIndex%2
	}
}
