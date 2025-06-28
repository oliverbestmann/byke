package main

import (
	"bytes"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/pkg/profile"
	ecs "gobevy"
	"math"
	"math/rand/v2"
)

func main() {
	defer profile.Start(profile.CPUProfile).Stop()

	var app ecs.App

	app.AddPlugin(Plugin)

	app.AddSystems(Startup, setupObjectsSystem)
	app.AddSystems(Update, movementSystem, blinkSystem)
	app.AddSystems(Update, followMouseSystem)

	app.Run()
}

type BlinkFrequency struct {
	ecs.Component[BlinkFrequency]
	Value float64
}

func setupObjectsSystem(commands *ecs.Commands, screenSize ScreenSize) {
	gopher, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(GopherPNG))

	randVec := func() Vec {
		for {
			vec := Vec{
				X: rand.Float64()*2 - 1,
				Y: rand.Float64()*2 - 1,
			}

			if vec.Length() <= 1 {
				return vec
			}
		}
	}

	for idx := range 50 {
		size := rand.Float64()*32 + 16

		commands.Spawn(
			ecs.Name("Gopher"),
			Velocity{
				Vec: randVec().Mul(50),
			},
			Transform{
				Translation: randVec().MulEach(Vec(screenSize)),
				Scale:       VecOf(1.0, 1.0),
			},
			Size{
				Vec: VecOf(size, size),
			},
			Layer{
				Z: float64(idx),
			},
			BlinkFrequency{
				Value: rand.Float64() + 0.5,
			},
			Sprite{
				Image: gopher,
			})
	}
}

type Velocity struct {
	ecs.Component[Velocity]
	Vec
}

var _ = ecs.ValidateComponent[Velocity]()

type MovementValues struct {
	EntityId ecs.EntityId
	Name     ecs.Name

	Velocity  Velocity
	Transform *Transform
}

func movementSystem(query ecs.Query[MovementValues], vt VirtualTime) {
	for item := range query.Items() {
		item.Transform.Translation.X += item.Velocity.X * vt.DeltaSecs
		item.Transform.Translation.Y += item.Velocity.Y * vt.DeltaSecs
		item.Transform.Rotation += Rad(vt.DeltaSecs)
	}
}

type FollowMouseValues struct {
	Velocity  *Velocity
	Transform Transform
}

func followMouseSystem(query ecs.Query[FollowMouseValues], cursor MouseCursor, vt VirtualTime) {
	for res := range query.Items() {
		dir := Vec(cursor).Sub(res.Transform.Translation).Normalized()
		res.Velocity.Vec = res.Velocity.Add(dir.Mul(200 * vt.DeltaSecs))
	}
}

type BlinkValues struct {
	ColorTint *ColorTint
	Frequency BlinkFrequency
}

func blinkSystem(query ecs.Query[BlinkValues], time VirtualTime) {
	for item := range query.Items() {
		alpha := math.Abs(math.Sin(time.Elapsed.Seconds() / item.Frequency.Value * math.Pi * 2))
		item.ColorTint.A = float32(alpha)
	}
}
