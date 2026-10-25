package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Adapted from https://bevy.org/examples/2d-rendering/pixel-grid-snap/
// Renders a pixel art scene to a small canvas and scales it up to the window.
// The rotating square on the right is drawn at full resolution for comparison.

const canvasWidth, canvasHeight = 160, 90

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, ctx *RenderContext, assets *Assets) {
	canvas := NewTexture2d(ctx, NewTexture2dOptions{
		Label:        "Canvas",
		Format:       wgpu.TextureFormatRGBA8UnormSrgb,
		Width:        canvasWidth,
		Height:       canvasHeight,
		MipmapLevels: 1,

		// nearest filtering keeps the pixels sharp when the canvas is scaled up
		SamplerConfig: SamplerConfig{FilterMode: wgpu.FilterModeNearest},
	})

	// this camera renders the pixel art scene into the canvas, one unit is one pixel
	commands.Spawn(
		Camera{Order: -1},
		Camera2d,
		RenderTarget{Texture: AsRenderTexture(canvas)},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
		},
	)

	commands.Spawn(
		TransformFromXY(-40, 0),
		Sprite{Texture: assets.TextureWithSettings("rocket.png", &LoadTextureSettings{
			FilterMode: wgpu.FilterModeNearest,
		}).Await()},
		shared.RotatingZ{Speed: 0.5},
	)

	commands.Spawn(
		TransformFromXY(10, 0),
		MeshInstance{Mesh: Rectangle(glm.Vec2f{24, 24})},
		ColorMaterial{Color: ColorSRGB(0.9, 0.4, 0.2)},
		shared.RotatingZ{Speed: 0.5},
	)

	// the window camera shows the canvas and a high resolution square
	commands.Spawn(
		Camera{},
		Camera2d,
		RenderLayersOf(1),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixed{Viewport: glm.Vec2f{canvasWidth, canvasHeight}},
		},
	)

	commands.Spawn(
		Sprite{Texture: canvas},
		RenderLayersOf(1),
	)

	commands.Spawn(
		TransformFromXYZ(50, 0, 0.1),
		MeshInstance{Mesh: Rectangle(glm.Vec2f{24, 24})},
		ColorMaterial{Color: ColorSRGB(0.2, 0.6, 0.9)},
		RenderLayersOf(1),
		shared.RotatingZ{Speed: 0.5},
	)
}
