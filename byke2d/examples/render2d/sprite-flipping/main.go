package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Adapted from https://bevy.org/examples/2d-rendering/sprite-flipping/
// Shows a sprite as is, flipped on the x axis, on the y axis and on both.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(Camera{})

	texture := assets.TextureWithSettings("marker.png", &LoadTextureSettings{
		FilterMode: wgpu.FilterModeNearest,
	}).Await()

	sprites := []Sprite{
		{Texture: texture},
		{Texture: texture, FlipX: true},
		{Texture: texture, FlipY: true},
		{Texture: texture, FlipX: true, FlipY: true},
	}

	for idx, sprite := range sprites {
		sprite.CustomSize = Some(glm.Vec2f{128, 128})

		commands.Spawn(
			TransformFromXY(float32(idx)*150-225, 0),
			sprite,
		)
	}
}
