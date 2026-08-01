package main

import (
	_ "image/png"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		FirstPersonViewController{Pitch: -0.5, Yaw: -0.25},
		PerspectiveProjection{Fov: glm.DegToRad(50), Near: 0.1},
		TransformFromXYZ(-2.5, 4.5, 9.0),
	)

	commands.Spawn(
		TransformFromXYZ(0, 0.5, 0),
		MeshInstance{Mesh: Cube()},
		StandardMaterial{BaseColor: ColorSRGB(0.5, 0.6, 1.0), PerceptualRoughness: 0.8},
	)

	commands.Spawn(
		NewTransform().WithRotationX(glm.DegToRad(-90)),
		MeshInstance{Mesh: Circle(4.0, 64)},
		StandardMaterial{
			BaseColor:           ColorSRGB(1, 1, 1),
			DoubleSided:         true,
			PerceptualRoughness: 0.5,
			Metallic:            0.5,
		},
	)

	commands.Spawn(
		TransformFromXYZ(4, 8, 4),
		PointLight{
			Color:       ColorLinearRGB(1, 1, 1),
			Illuminance: 50,
		},
	)
}
