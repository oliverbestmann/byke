package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
)

// Adapted from https://bevy.org/examples/2d-rendering/move-sprite/
// Moves a sprite up and down.

var _ = ValidateComponent[Direction]()

type Direction struct {
	Component[Direction]
	Up bool
}

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, spriteMovementSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 100})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(Camera{})

	commands.Spawn(
		Sprite{Texture: assets.Texture("rocket.png").Await()},
		NewTransform(),
		Direction{Up: true},
	)
}

// The sprite moves 150 units per second and turns around at the top and bottom.
func spriteMovementSystem(
	vt VirtualTime,
	query Query[struct {
		Direction *Direction
		Transform *Transform
	}],
) {
	for item := range query.Items() {
		step := 150 * vt.DeltaSecs
		if !item.Direction.Up {
			step = -step
		}

		item.Transform.Translation[1] += step

		switch {
		case item.Transform.Translation[1] > 200:
			item.Direction.Up = false
		case item.Transform.Translation[1] < -200:
			item.Direction.Up = true
		}
	}
}
