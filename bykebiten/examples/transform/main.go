package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	. "github.com/oliverbestmann/byke/gm"
)

//go:embed gopher.png
var GopherPNG []byte

func main() {
	var app App

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

func createGopherSystem(commands *Commands, screenSize ScreenSize) {
	gopher, _, _ := ebitenutil.NewImageFromReader(bytes.NewReader(GopherPNG))

	commands.Spawn(
		NewTransform().WithTranslation(screenSize.Mul(0.5)),
		Gopher{},
		Sprite{Image: gopher},
		AnchorCenter,
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
