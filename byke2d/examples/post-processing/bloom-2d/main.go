package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/2d-rendering/bloom-2d/
// Colors brighter than 1 glow on HDR cameras.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		Camera2d,
		// HDR enables bloom and tonemapping
		HDR{},
		ClearColor{Color: ColorBlack},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeAutoMin{MinWidth: 800, MinHeight: 600},
		},
	)

	commands.Spawn(
		Sprite{
			Texture:    assets.Texture("marker.png").Await(),
			CustomSize: Some(glm.Vec2f{256, 256}),
			Color:      ColorLinearRGB(5, 5, 5),
		},
	)

	commands.Spawn(
		TransformFromXY(-250, 0),
		MeshInstance{Mesh: Circle(100, 64)},
		ColorMaterial{Color: ColorLinearRGB(7.5, 0, 7.5)},
	)

	commands.Spawn(
		TransformFromXY(250, 0),
		MeshInstance{Mesh: RegularPolygon(100, 6)},
		ColorMaterial{Color: ColorLinearRGB(6.25, 9.4, 9.1)},
	)
}
