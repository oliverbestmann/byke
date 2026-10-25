package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/3d-rendering/parenting/
// A child entity follows the transform of its parent.

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 50})
}

func setupSystem(commands *Commands) {
	cube := CubeWithSize(glm.Vec3f{2, 2, 2})
	material := StandardMaterial{BaseColor: ColorSRGB(0.8, 0.7, 0.6), PerceptualRoughness: 0.5}

	// the parent cube rotates around the x axis
	commands.Spawn(
		TransformFromXYZ(0, 0, 1),
		MeshInstance{Mesh: cube},
		material,
		shared.RotatingX{Speed: 3},

		// the child is positioned relative to the parent and rotates with it
		SpawnChild(
			TransformFromXYZ(0, 0, 3),
			MeshInstance{Mesh: cube},
			material,
		),
	)

	commands.Spawn(
		TransformFromXYZ(4, 5, -4),
		DefaultPointLight,
	)

	commands.Spawn(
		Camera3d,
		Camera{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(5, 10, 10).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)
}
