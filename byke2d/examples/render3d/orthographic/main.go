package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/3d-rendering/orthographic/

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
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			// 6 world units fill the height of the window
			ScalingMode: ScalingModeFixedVertical{ViewportHeight: 6},
			Far:         1000,
		},
		TransformFromXYZ(5, 5, 5).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)

	// plane
	commands.Spawn(
		MeshInstance{Mesh: PlaneWithSize(glm.Vec2f{5, 5})},
		StandardMaterial{BaseColor: ColorSRGB(0.3, 0.5, 0.3), PerceptualRoughness: 0.5},
	)

	// cubes
	cube := Cube()
	material := StandardMaterial{BaseColor: ColorSRGB(0.8, 0.7, 0.6), PerceptualRoughness: 0.5}

	for _, pos := range []glm.Vec3f{{1.5, 0.5, 1.5}, {1.5, 0.5, -1.5}, {-1.5, 0.5, 1.5}, {-1.5, 0.5, -1.5}} {
		commands.Spawn(
			TransformFromXYZ(pos.XYZ()),
			MeshInstance{Mesh: cube},
			material,
		)
	}

	// light
	commands.Spawn(
		TransformFromXYZ(3, 8, 5),
		DefaultPointLight,
	)
}
