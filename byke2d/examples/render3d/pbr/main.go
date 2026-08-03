package main

import (
	_ "image/png"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	opts := &LoadTextureSettings{
		OverrideTextureViewDimension: wgpu.TextureViewDimensionCube,
	}

	specular := assets.TextureWithSettings("pisa_specular_rgb9e5_zstd.ktx2", opts)
	diffuse := assets.TextureWithSettings("pisa_diffuse_rgb9e5_zstd.ktx2", opts)

	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},

		ClearColor{Color: ColorSRGB(0.2, 0.2, 0.2)},

		TransformFromXYZ(0, 0, 100),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 12},
			Near:           1,
			Far:            -101,
		},

		// DefaultPerspectiveProjection,
		// FirstPersonViewController{},

		EnvironmentMapLight{
			DiffuseMap:  diffuse.Await(),
			SpecularMap: specular.Await(),
			Intensity:   900,
		},
	)

	commands.Spawn(
		TransformFromXYZ(-50, -50, -50).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
		DirectionalLight{Illuminance: 1_500},
	)

	sphere := Sphere(0.45, 64, 32)

	for y := -2; y <= 2; y++ {
		for x := -5; x <= 5; x++ {
			x01 := float32(x+5) / 10.0
			y01 := float32(y+2) / 4.0

			commands.Spawn(
				TransformFromXYZ(float32(x), float32(y), 0),
				MeshInstance{Mesh: sphere},
				StandardMaterial{
					BaseColor:           ColorSRGB(1.0, 0.85, 0.56),
					Metallic:            y01,
					PerceptualRoughness: x01,
				},
			)
		}
	}

}
