package main

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/pkg/profile"
	"math"
	"math/rand/v2"
)

func main() {
	defer profile.Start(profile.MemProfile).Stop()

	var app byke.App

	app.AddPlugin(Plugin)

	app.AddSystems(byke.Startup, setupObjectsSystem)

	app.AddSystems(byke.FixedUpdate, byke.System(followMouseSystem, movementSystem).Chain())

	app.AddSystems(byke.Update, blinkSystem)

	app.InitState(byke.StateType[PauseState]{})

	app.AddSystems(byke.Update, togglePauseState, rotateSystem)

	app.AddSystems(byke.OnEnter(PauseStatePaused), pausedSystem)
	app.AddSystems(byke.OnExit(PauseStatePaused), unpausedSystem)

	fmt.Println(app.Run())
}

type BlinkFrequency struct {
	byke.ComparableComponent[BlinkFrequency]
	Value float64
}

func setupObjectsSystem(commands *byke.Commands, screenSize ScreenSize) {
	gopher, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(GopherPNG))

	randVec := func() gm.Vec {
		for {
			vec := gm.Vec{
				X: rand.Float64()*2 - 1,
				Y: rand.Float64()*2 - 1,
			}

			if vec.Length() <= 1 {
				return vec
			}
		}
	}

	commands.Spawn(
		byke.Named("Sun"),

		Rotate{
			AngularVelocity: 1,
		},

		Transform{
			Translation: gm.Vec{X: 400, Y: 300},
			Scale:       gm.Vec{X: 1, Y: 1},
			Rotation:    0,
		},
		Size{
			Vec: gm.Vec{X: 50, Y: 50},
		},
		Sprite{
			Image: gopher,
		},

		byke.SpawnChild(
			Transform{
				Translation: gm.Vec{X: 200},
				Scale:       gm.Vec{X: 1, Y: 2},
				Rotation:    math.Pi / 2,
			},
			Size{
				Vec: gm.Vec{X: 25, Y: 25},
			},

			Sprite{
				Image: gopher,
			},

			Rotate{
				//	AngularVelocity: -1,
			},
		),
	)

	return

	for idx := range 50 {
		size := rand.Float64()*32 + 16

		commands.Spawn(
			byke.Named("Gopher"),

			Transform{
				Translation: randVec().MulEach(screenSize.Vec),
				Scale:       gm.VecOf(1.0, 1.0),
			},

			byke.SpawnChild(
				Velocity{
					Vec: randVec().Mul(50),
				},
				NewTransform(),
				Size{
					Vec: gm.VecOf(size, size),
				},
				Layer{
					Z: float64(idx),
				},
				BlinkFrequency{
					Value: rand.Float64() + 0.5,
				},
				Sprite{
					Image: gopher,
				},
			),
		)
	}
}

type Velocity struct {
	byke.ComparableComponent[Velocity]
	gm.Vec
}

var _ = byke.ValidateComponent[Velocity]()

type MovementValues struct {
	EntityId byke.EntityId
	Name     byke.Name

	Velocity  Velocity
	Transform *Transform
}

func movementSystem(query byke.Query[MovementValues], vt byke.VirtualTime) {
	for item := range query.Items() {
		item.Transform.Translation.X += item.Velocity.X * vt.DeltaSecs
		item.Transform.Translation.Y += item.Velocity.Y * vt.DeltaSecs
		item.Transform.Rotation += gm.Rad(vt.DeltaSecs)
	}
}

type FollowMouseValues struct {
	Velocity  *Velocity
	Transform Transform
}

func followMouseSystem(query byke.Query[FollowMouseValues], cursor MouseCursor, vt byke.VirtualTime) {
	for res := range query.Items() {
		dir := cursor.Vec.Sub(res.Transform.Translation).Normalized()
		res.Velocity.Vec = res.Velocity.Add(dir.Mul(200 * vt.DeltaSecs))
	}
}

type BlinkValues struct {
	ColorTint *ColorTint
	Frequency BlinkFrequency
}

func blinkSystem(query byke.Query[BlinkValues], time byke.VirtualTime) {
	for item := range query.Items() {
		alpha := math.Abs(math.Sin(time.Elapsed.Seconds() / item.Frequency.Value * math.Pi * 2))
		green := math.Abs(math.Sin(time.Elapsed.Seconds() / item.Frequency.Value * math.Pi * 2.1))

		item.ColorTint.R = float32(green)*0.75 + 0.25
		item.ColorTint.B = float32(green)*0.75 + 0.25
		item.ColorTint.A = float32(alpha)*0.75 + 0.25
	}
}

type PauseState int

const PauseStateRunning PauseState = 0
const PauseStatePaused PauseState = 1

func togglePauseState(
	state byke.State[PauseState],
	nextState *byke.NextState[PauseState],
	keys Keys,
) {
	if keys.IsJustPressed(ebiten.KeyEscape) {
		isRunning := state.Current() == PauseStateRunning

		if isRunning {
			nextState.Set(PauseStatePaused)
		} else {
			nextState.Set(PauseStateRunning)
		}
	}
}

func pausedSystem(
	commands *byke.Commands,
	vt *byke.VirtualTime,
	screenSize ScreenSize,
) {
	vt.Scale = 0.0

	image, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(EbitenPNG))
	commands.Spawn(
		byke.DespawnOnExitState(PauseStatePaused),

		Transform{
			Translation: screenSize.Mul(0.5),
			Scale:       gm.VecOf(1.0, 1.0),
		},

		Layer{Z: math.Inf(1)},

		Sprite{
			Image: image,
		},
	)
}

func unpausedSystem(
	vt *byke.VirtualTime,
) {
	vt.Scale = 1.0
}

func rotateSystem(
	query byke.Query[struct {
	Rotate    Rotate
	Transform *Transform
}],
	vt byke.VirtualTime,
) {
	for item := range query.Items() {
		item.Transform.Rotation += gm.Rad(vt.DeltaSecs) * item.Rotate.AngularVelocity
		// item.Transform.Scale.X = math.Sin(vt.Elapsed.Seconds()) + 1.1
	}
}

var _ = byke.ValidateComponent[Rotate]()

type Rotate struct {
	byke.Component[Rotate]
	AngularVelocity gm.Rad
}
