package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Textures get mipmaps when they are loaded, so scaled down sprites
// stay smooth instead of flickering.

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
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1100},
		},
	)

	texture := assets.Texture("input.jpg").Await()

	// the texture is 512 pixels wide, draw it at half the size of the previous one
	x := float32(-544)
	for size := float32(512); size >= 16; size /= 2 {
		commands.Spawn(
			TransformFromXY(x+size/2, 0),
			Sprite{Texture: texture, CustomSize: Some(glm.Vec2f{size, size})},
		)

		x += size + 16
	}
}
