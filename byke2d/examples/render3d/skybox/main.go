package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// A skybox and an environment map light, reflected by spheres of
// different roughness. Move the camera with the mouse and WASD.

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint(shared.HintFirstPerson))
	app.InsertResource(GlobalAmbientLightNone)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	opts := &LoadTextureSettings{OverrideTextureViewDimension: wgpu.TextureViewDimensionCube}
	diffuse := assets.TextureWithSettings("pisa_diffuse_rgb9e5_zstd.ktx2", opts)
	specular := assets.TextureWithSettings("pisa_specular_rgb9e5_zstd.ktx2", opts)

	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		FirstPersonViewController{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(0, 1, 8).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),

		// the specular map is sharp enough to be used as the sky
		Skybox{Texture: specular.Await(), Intensity: ColorLinearRGB(1, 1, 1)},
		EnvironmentMapLight{
			DiffuseMap:  diffuse.Await(),
			SpecularMap: specular.Await(),
			Intensity:   900,
		},
	)

	sphere := Sphere(0.5, 64, 32)

	// metallic spheres on top, dielectric ones below, roughness increases to the right
	for idx := range 5 {
		roughness := float32(idx) / 4
		x := float32(idx-2) * 1.25

		commands.Spawn(
			TransformFromXYZ(x, 0.65, 0),
			MeshInstance{Mesh: sphere},
			StandardMaterial{BaseColor: ColorSRGB(1, 0.85, 0.6), Metallic: 1, PerceptualRoughness: roughness},
		)

		commands.Spawn(
			TransformFromXYZ(x, -0.65, 0),
			MeshInstance{Mesh: sphere},
			StandardMaterial{BaseColor: ColorSRGB(0.8, 0.1, 0.1), PerceptualRoughness: roughness},
		)
	}
}
