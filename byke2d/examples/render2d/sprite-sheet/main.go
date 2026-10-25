package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Adapted from https://bevy.org/examples/2d-rendering/sprite-sheet/
// Animates a sprite by stepping through the tiles of a texture atlas.

var _ = ValidateComponent[AnimationTimer]()

type AnimationTimer struct {
	Component[AnimationTimer]
	Timer
}

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, animateSpriteSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 20, 30})
}

func setupSystem(commands *Commands, assets *Assets) {
	// nearest filtering keeps the pixel art sharp
	texture := assets.TextureWithSettings("figure.png", &LoadTextureSettings{
		FilterMode: wgpu.FilterModeNearest,
	}).Await()

	// the walk animation: 8 tiles of 32x32 pixels in the second row
	layout := TextureAtlasLayoutFromGrid(GridOptions{
		StartRow: 1,
		Count:    8,
		Width:    32,
		Height:   32,
	})

	commands.Spawn(Camera{})

	commands.Spawn(
		NewTransform().WithScaleXY(6, 6),
		Sprite{Texture: texture},
		TextureAtlas{Layout: layout, Wrapping: true},
		AnimationTimer{Timer: NewTimerWithFrequency(10)},
	)
}

func animateSpriteSystem(
	vt VirtualTime,
	query Query[struct {
		Timer        *AnimationTimer
		TextureAtlas *TextureAtlas
	}],
) {
	for item := range query.Items() {
		if item.Timer.Tick(vt.Delta).JustFinished() {
			item.TextureAtlas.Index += 1
		}
	}
}
