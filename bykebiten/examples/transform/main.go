package main

import (
	"embed"
	_ "embed"
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	. "github.com/oliverbestmann/byke/gm"
)

//go:embed assets
var assets embed.FS

func main() {
	var app App

	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(GamePlugin)

	// optional: configure the game window
	app.InsertResource(WindowConfig{
		Title:  "Example",
		Width:  800,
		Height: 600,
	})

	app.AddSystems(Startup, createGopherSystem)
	app.AddSystems(Update, rotateGopherSystem)

	fmt.Println(app.Run())
}

var _ = ValidateComponent[Gopher]()

// Gopher is a component that identifies an entity as a gopher.
type Gopher struct {
	ComparableComponent[Gopher]
}

func createGopherSystem(commands *Commands, assets *Assets) {
	rectImage := assets.Image("rect.png").Await()

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 100, Y: 100}),
		Gopher{},
		Sprite{Image: rectImage},
		AnchorTopLeft,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 200, Y: 100}),
		Gopher{},
		Sprite{Image: rectImage},
		AnchorCenter,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 300, Y: 100}),
		Gopher{},
		Sprite{Image: rectImage},
		AnchorBottomRight,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 100, Y: 200}),
		Gopher{},
		Sprite{Image: rectImage, FlipX: true},
		AnchorTopLeft,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 200, Y: 200}),
		Gopher{},
		Sprite{Image: rectImage, FlipX: true},
		AnchorCenter,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 300, Y: 200}),
		Gopher{},
		Sprite{Image: rectImage, FlipX: true},
		AnchorBottomRight,
	)

	// with custom size

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 100, Y: 300}),
		Gopher{},
		Sprite{Image: rectImage, FlipX: true, CustomSize: Some(VecSplat(64.0))},
		AnchorTopLeft,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 300, Y: 300}),
		Gopher{},
		Sprite{Image: rectImage, FlipX: true, CustomSize: Some(VecSplat(64.0))},
		AnchorCenter,
	)

	commands.Spawn(
		NewTransform().WithScale(VecSplat(2.0)).WithTranslation(Vec{X: 500, Y: 300}),
		Gopher{},
		Sprite{Image: rectImage, FlipX: true, CustomSize: Some(VecSplat(64.0))},
		AnchorBottomRight,
	)
}

type rotateGopherItems struct {
	With[Gopher]

	// components we want to mutate must be pointers
	Transform *Transform
}

func rotateGopherSystem(gophers Query[rotateGopherItems], t VirtualTime) {
	for gopher := range gophers.Items() {
		// rotate gopher around its center
		gopher.Transform.Rotation += Rad(2 * t.DeltaSecs)
	}
}
