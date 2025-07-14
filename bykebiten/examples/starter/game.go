package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	. "github.com/oliverbestmann/byke/gm"
)

var _ = ValidateComponent[Ducky]()

func pluginGame(app *App) {
	app.AddSystems(OnEnter(ScreenGame), spawnGameScreenSystem)

	app.AddSystems(Update, System(inputToMotionIntentSystem, movementSystem, updateAnimation).Chain())
}

type Ducky struct {
	Component[Ducky]
	AnimationTimer Timer
	AnimationIndex int
	Speed          float64
}

type MotionIntent struct {
	Component[MotionIntent]
	Direction Vec
}

func spawnGameScreenSystem(commands *Commands) {
	commands.Spawn(
		DespawnOnExitState(ScreenGame),
		TransformFromXY(400, 300),
		Ducky{
			AnimationTimer: NewTimerFromSeconds(0.25, TimerModeRepeating),
			Speed:          80,
		},
		MotionIntent{},
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

func inputToMotionIntentSystem(
	keys Keys,
	query Query[struct {
		MotionIntent *MotionIntent
	}],
) {
	for item := range query.Items() {
		dir := &item.MotionIntent.Direction
		dir.X = 0
		dir.Y = 0

		if keys.IsPressed(ebiten.KeyLeft) {
			dir.X -= 1
		}
		if keys.IsPressed(ebiten.KeyRight) {
			dir.X += 1
		}
		if keys.IsPressed(ebiten.KeyUp) {
			dir.Y -= 1
		}
		if keys.IsPressed(ebiten.KeyDown) {
			dir.Y += 1
		}
	}
}

func movementSystem(
	vt VirtualTime,
	query Query[struct {
		Ducky        Ducky
		MotionIntent *MotionIntent
		Transform    *Transform
	}],
) {
	for item := range query.Items() {
		delta := item.MotionIntent.Direction.Mul(item.Ducky.Speed * vt.DeltaSecs)
		item.Transform.Translation = item.Transform.Translation.Add(delta)
	}
}

func updateAnimation(vt VirtualTime, query Query[struct {
	MotionIntent MotionIntent
	Ducky        *Ducky
	TileIndex    *TileIndex
}]) {
	item, ok := query.Single()
	if !ok {
		return
	}

	// update the frame counter
	item.Ducky.AnimationIndex += item.Ducky.AnimationTimer.Tick(vt.Delta).TimesFinishedThisTick()

	walking := item.MotionIntent.Direction.LengthSqr() > 0

	// update depending on animation state
	//goland:noinspection GoDfaConstantCondition
	switch {
	case walking:
		item.TileIndex.Index = 6 + (item.Ducky.AnimationIndex-6)%6

	case !walking:
		item.TileIndex.Index = 0 + item.Ducky.AnimationIndex%2
	}
}
