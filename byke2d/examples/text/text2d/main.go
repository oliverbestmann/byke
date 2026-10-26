package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Draws text in 2d. The anchor decides which point of the text sits at its position.
// All four texts are placed at the dot in the center.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
		},
	)

	commands.Spawn(
		MeshInstance{Mesh: Circle(4, 16)},
		ColorMaterial{Color: ColorWhite},
	)

	anchored := []struct {
		Anchor *Anchor
		Text   string
		Color  Color
	}{
		{AnchorTopLeft, "Top left", ColorSRGB(1, 0.5, 0.5)},
		{AnchorTopRight, "Top right", ColorSRGB(0.5, 1, 0.5)},
		{AnchorBottomLeft, "Bottom left", ColorSRGB(0.5, 0.7, 1)},
		{AnchorBottomRight, "Bottom right", ColorSRGB(1, 1, 0.5)},
	}

	for _, item := range anchored {
		commands.Spawn(
			TransformFromXYZ(0, 0, 0.1),
			item.Anchor,
			Text{Text: item.Text, Size: 32, Color: item.Color},
		)
	}

	commands.Spawn(
		TransformFromXYZ(0, 150, 0.1),
		AnchorCenter,
		Text{Text: "Text in 2d\nwith multiple lines", Size: 40, Color: ColorWhite},
	)
}
