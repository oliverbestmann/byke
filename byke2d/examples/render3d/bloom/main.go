package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/3d-rendering/bloom-3d/
// Bright emissive materials glow on HDR cameras.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera3d,
		Camera{},
		// HDR enables bloom and tonemapping
		HDR{},
		ClearColor{Color: ColorBlack},
		DefaultPerspectiveProjection,
		TransformFromXYZ(-2, 2.5, 5).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)

	// emissive values above 1 are what makes the material glow
	materials := []StandardMaterial{
		{BaseColor: ColorWhite, EmissiveScale: glm.Vec3f{13.99, 5.32, 2.0}},
		{BaseColor: ColorWhite, EmissiveScale: glm.Vec3f{2.0, 13.99, 5.32}},
		{BaseColor: ColorWhite, EmissiveScale: glm.Vec3f{5.32, 2.0, 13.99}},
	}

	nonEmissive := StandardMaterial{BaseColor: ColorSRGB(0.5, 0.5, 0.5), PerceptualRoughness: 0.5}

	sphere := Sphere(0.4, 32, 16)

	for x := -5; x < 5; x++ {
		for z := -5; z < 5; z++ {
			// pick a material that looks random, but is the same every run
			material := nonEmissive
			if idx := (x*7 + z*13 + 100) % 6; idx < len(materials) {
				material = materials[idx]
			}

			commands.Spawn(
				TransformFromXYZ(float32(x)*2, 0, float32(z)*2),
				MeshInstance{Mesh: sphere},
				material,
			)
		}
	}
}
