package main

import (
	"embed"
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	. "github.com/oliverbestmann/byke/gm"
	"github.com/pkg/profile"
	"math"
	"math/rand/v2"
)

//go:embed assets
var assets embed.FS

func main() {
	defer profile.Start(profile.CPUProfile).Stop()

	var app App

	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(GamePlugin)

	// optional: configure the game window
	app.InsertResource(WindowConfig{
		Title:  "Example",
		Width:  800,
		Height: 600,
	})

	app.AddSystems(Startup, createCamera)
	app.AddSystems(Startup, createSprites)
	app.AddSystems(Update, System(avoidCursorSystem, movementSystem, dampenSystem, wrapScreenSystem).Chain())

	app.AddSystems(Update, System(pauseSystem).RunIf(KeyJustPressed(ebiten.KeySpace)))

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

var worldSize = Vec{X: 800, Y: 600}

func createCamera(commands *Commands) {
	commands.Spawn(
		Camera{},

		OrthographicProjection{
			Scale: 1.0,
			ScalingMode: ScalingModeAutoMin{
				MinWidth:  worldSize.X,
				MinHeight: worldSize.Y,
			},
			ViewportOrigin: VecSplat(0.5),
		},
	)
}

func createSprites(commands *Commands, assets *Assets) {
	image := assets.Image("ebiten.png").Await()

	for range 1000 {
		posX := (rand.Float64() - 0.5) * worldSize.X
		posY := (rand.Float64() - 0.5) * worldSize.Y

		velX := (rand.Float64() - 0.5) * 30
		velY := (rand.Float64() - 0.5) * 30
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

func dampenSystem(t VirtualTime, items Query[*Velocity]) {
	for item := range items.Items() {
		if item.Linear.LengthSqr() < 10 {
			continue
		}
		item.Linear = item.Linear.Mul(0.999 * (1 - t.DeltaSecs))
	}
}

type wrapScreenItem struct {
	With[WrapScreen]

	Transform *Transform
}

func wrapScreenSystem(items Query[wrapScreenItem]) {
	for item := range items.Items() {
		pos := item.Transform.Translation.Add(worldSize.Mul(0.5))

		pos.X = math.Mod(pos.X, worldSize.X)
		pos.Y = math.Mod(pos.Y, worldSize.Y)

		item.Transform.Translation = pos.Sub(worldSize.Mul(0.5))
	}
}

func avoidCursorSystem(mouseCursor MouseCursor, vt VirtualTime, screenSize ScreenSize,
	items Query[struct {
		_         With[AvoidCursor]
		Velocity  *Velocity
		Transform Transform
	}],
	cameras Query[struct {
		Projection OrthographicProjection
		Transform  GlobalTransform
	}],
) {
	proj := cameras.MustFirst()
	toScreen := CalculateWorldToScreenTransform(proj.Projection, proj.Transform, screenSize.Vec)
	toWorld, ok := toScreen.TryInverse()
	if !ok {
		return
	}

	worldCursor := toWorld.Transform(mouseCursor.Vec)

	for item := range items.Items() {
		pos := item.Transform.Translation

		if worldCursor.DistanceTo(pos) > 100 {
			continue
		}

		// TODO use time independent exponential interpolation here
		f := 10 * 200 / worldCursor.DistanceTo(pos)

		newVelocity := item.Velocity.Linear.Mul(1 - vt.DeltaSecs).Add(worldCursor.VecTo(pos).Normalized().Mul(f * vt.DeltaSecs))
		item.Velocity.Linear = newVelocity
	}
}

func pauseSystem(vt *VirtualTime) {
	if vt.Scale == 0 {
		vt.Scale = 1
	} else {
		vt.Scale = 0
	}
}
