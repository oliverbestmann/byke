package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Renders the same shape into two small textures, the left one with MSAA.
// Both are shown enlarged, so the smooth and the jagged edges are easy to see.

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, ctx *RenderContext) {
	commands.Spawn(
		TransformFromXY(0, 0).WithRotationZ(0.3),
		MeshInstance{Mesh: RegularPolygon(24, 5)},
		ColorMaterial{Color: ColorWhite},
		shared.RotatingZ{Speed: 0.2},
	)

	// one camera with MSAA and one without, each renders into its own texture
	withMSAA := spawnTextureCamera(commands, ctx, true)
	withoutMSAA := spawnTextureCamera(commands, ctx, false)

	// the window camera shows the two textures side by side
	commands.Spawn(
		Camera{Order: 1},
		Camera2d,
		RenderLayersOf(1),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 140},
		},
	)

	commands.Spawn(
		TransformFromXY(-33, 0),
		Sprite{Texture: withMSAA},
		RenderLayersOf(1),
	)

	commands.Spawn(
		TransformFromXY(33, 0),
		Sprite{Texture: withoutMSAA},
		RenderLayersOf(1),
	)
}

func spawnTextureCamera(commands *Commands, ctx *RenderContext, msaa bool) *Texture {
	texture := NewTexture2d(ctx, NewTexture2dOptions{
		Format:       wgpu.TextureFormatRGBA8UnormSrgb,
		Width:        64,
		Height:       64,
		MipmapLevels: 1,

		// nearest filtering keeps the pixels sharp when the texture is enlarged
		SamplerConfig: SamplerConfig{FilterMode: wgpu.FilterModeNearest},
	})

	camera := commands.Spawn(
		Camera{},
		Camera2d,
		RenderTarget{Texture: AsRenderTexture(texture)},
		// the default 2d depth range ends at the camera, move it behind the shape
		TransformFromXYZ(0, 0, -0.5),
		OrthographicProjection{ViewportOrigin: glm.Vec2f{0.5, 0.5}, ScalingMode: ScalingModeWindowSize{}},
	)

	if msaa {
		camera.Insert(MSAA{})
	}

	return texture
}
