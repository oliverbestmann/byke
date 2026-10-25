package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Adapted from https://bevy.org/examples/3d-rendering/render-to-texture/
// A first camera renders a cube into a texture, which is then used as
// the texture of a second cube rendered by the main camera.

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 50})
}

func setupSystem(commands *Commands, ctx *RenderContext) {
	image := NewTexture2d(ctx, NewTexture2dOptions{
		Label:        "render target",
		Format:       wgpu.TextureFormatRGBA8UnormSrgb,
		Width:        512,
		Height:       512,
		MipmapLevels: 1,
	})

	cube := CubeWithSize(glm.Vec3f{4, 4, 4})

	// the cube rendered into the texture, only visible to the first camera
	commands.Spawn(
		MeshInstance{Mesh: cube},
		StandardMaterial{BaseColor: ColorSRGB(0.8, 0.7, 0.6), PerceptualRoughness: 0.5},
		RenderLayersOf(1),
		shared.RotatingX{Speed: 1.5},
		shared.RotatingZ{Speed: 1.3},
	)

	commands.Spawn(
		TransformFromXYZ(0, 0, 10),
		PointLight{Color: ColorWhite, Intensity: 200_000, Range: 20},
	)

	// the first camera renders into the texture
	commands.Spawn(
		Camera3d,
		Camera{Order: -1},
		RenderTarget{Texture: AsRenderTexture(image)},
		ClearColor{Color: ColorWhite},
		RenderLayersOf(1),
		DefaultPerspectiveProjection,
		TransformFromXYZ(0, 0, 15).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)

	// the main cube shows the texture
	commands.Spawn(
		TransformFromXYZ(0, 0, 1.5).WithRotationX(-glm.DegToRad(36)),
		MeshInstance{Mesh: cube},
		StandardMaterial{
			BaseColor:                 ColorWhite,
			PerceptualRoughness:       0.5,
			StandardMaterialBindGroup: StandardMaterialBindGroup{Texture: image},
		},
		shared.RotatingX{Speed: 1.0},
		shared.RotatingY{Speed: 0.7},
	)

	// the main camera renders to the window
	commands.Spawn(
		Camera3d,
		Camera{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(0, 0, 15).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)
}
