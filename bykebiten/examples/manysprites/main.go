package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	. "github.com/oliverbestmann/byke/gm"
	"math"
	"math/rand/v2"
)

//go:embed ebiten.png
var EbitenPNG []byte

func main() {
	var app App

	app.AddPlugin(GamePlugin)

	// optional: configure the game window
	app.InsertResource(WindowConfig{
		Title:  "Example",
		Width:  800,
		Height: 600,
	})

	app.AddSystems(Startup, createSprites)
	app.AddSystems(Update, System(avoidCursorSystem, movementSystem, wrapScreenSystem).Chain())

	fmt.Println(app.Run())
}

var _ = ValidateComponent[Velocity]()
var _ = ValidateComponent[WrapScreen]()
var _ = ValidateComponent[AvoidCursor]()

type Velocity struct {
	ComparableComponent[Velocity]
	Linear  Vec
	Angular Rad
}

type WrapScreen struct {
	ComparableComponent[WrapScreen]
}

type AvoidCursor struct {
	ComparableComponent[AvoidCursor]
}

func createSprites(commands *Commands, screenSize ScreenSize) {
	image, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(EbitenPNG))

	for range 1000 {
		posX := rand.Float64() * screenSize.X
		posY := rand.Float64() * screenSize.Y

		velX := (rand.Float64() - 0.5) * 20
		velY := (rand.Float64() - 0.5) * 20
		velAngular := Rad(rand.Float64() - 0.5)

		commands.Spawn(
			TransformFromXY(posX, posY).WithScale(VecSplat(32.0/256.0)),
			Velocity{Linear: Vec{X: velX, Y: velY}, Angular: velAngular},
			Sprite{Image: image},
			ColorTint{Color: color.RGBA(1.0, 1.0, 1.0, 0.25)},
			WrapScreen{},
			AvoidCursor{},
			AnchorCenter,
		)
	}
}

type moveSpritesItem struct {
	Velocity  Velocity
	Transform *Transform
}

func movementSystem(items Query[moveSpritesItem], t VirtualTime) {
	for item := range items.Items() {
		delta := item.Velocity.Linear.Mul(t.DeltaSecs)
		item.Transform.Translation = item.Transform.Translation.Add(delta)
		item.Transform.Rotation += item.Velocity.Angular * Rad(t.DeltaSecs)
	}
}

type wrapScreenItem struct {
	With[WrapScreen]

	Transform *Transform
}

func wrapScreenSystem(items Query[wrapScreenItem], screenSize ScreenSize) {
	for item := range items.Items() {
		pos := item.Transform.Translation.Add(screenSize.Vec)

		item.Transform.Translation.X = math.Mod(pos.X, screenSize.X)
		item.Transform.Translation.Y = math.Mod(pos.Y, screenSize.Y)
	}
}

func avoidCursorSystem(mouseCursor MouseCursor, vt VirtualTime, items Query[struct {
	With[AvoidCursor]
	Velocity  *Velocity
	Transform Transform
}]) {
	for item := range items.Items() {
		pos := item.Transform.Translation

		if mouseCursor.DistanceTo(pos) > 100 {
			continue
		}

		// TODO use time independent exponential interpolation here
		f := 10 * 200 / mouseCursor.DistanceTo(pos)

		newVelocity := item.Velocity.Linear.Mul(1 - vt.DeltaSecs).Add(mouseCursor.VecTo(pos).Normalized().Mul(f * vt.DeltaSecs))
		item.Velocity.Linear = newVelocity
	}
}
