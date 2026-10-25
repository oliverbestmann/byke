package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/3d-rendering/lighting/
// Shows point, spot and directional lights together.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	// ground plane
	commands.Spawn(
		MeshInstance{Mesh: PlaneWithSize(glm.Vec2f{10, 10})},
		StandardMaterial{BaseColor: ColorWhite, PerceptualRoughness: 1},
	)

	// cube
	commands.Spawn(
		TransformFromXYZ(0, 0.5, 0),
		MeshInstance{Mesh: Cube()},
		StandardMaterial{BaseColor: ColorSRGB(1, 0.75, 0.8), PerceptualRoughness: 0.5},
	)

	// sphere
	commands.Spawn(
		TransformFromXYZ(1.5, 1, 1.5),
		MeshInstance{Mesh: Sphere(0.5, 32, 16)},
		StandardMaterial{BaseColor: ColorSRGB(0.2, 0.8, 0.2), PerceptualRoughness: 0.5},
	)

	// red point light
	commands.Spawn(
		TransformFromXYZ(1, 2, 0),
		PointLight{
			Color:     ColorSRGB(1, 0, 0),
			Intensity: 100_000,
			Range:     20,

			ShadowsEnabled:   true,
			ShadowDepthBias:  DefaultPointLight.ShadowDepthBias,
			ShadowNormalBias: DefaultPointLight.ShadowNormalBias,
		},
	)

	// green spot light, pointing down
	commands.Spawn(
		TransformFromXYZ(-1, 2, 0).LookingAt(glm.Vec3f{-1, 0, 0}, glm.Vec3f{0, 0, 1}),
		SpotLight{
			Color:      ColorSRGB(0, 1, 0),
			Intensity:  100_000,
			Range:      20,
			InnerAngle: 0.6,
			OuterAngle: 0.8,

			ShadowsEnabled:   true,
			ShadowDepthBias:  DefaultSpotLight.ShadowDepthBias,
			ShadowNormalBias: DefaultSpotLight.ShadowNormalBias,
		},
	)

	// blue point light
	commands.Spawn(
		TransformFromXYZ(0, 4, 0),
		PointLight{
			Color:     ColorSRGB(0, 0, 1),
			Intensity: 100_000,
			Range:     20,
		},
	)

	// directional light, as bright as an overcast day.
	// Its shadow volume starts at its position, so place it above the scene.
	commands.Spawn(
		TransformFromXYZ(0, 10, 10).WithRotationX(-glm.DegToRad(45)),
		DirectionalLight{
			Color:       ColorWhite,
			Illuminance: 1000,

			ShadowsEnabled:   true,
			ShadowVolume:     glm.Vec3f{20, 20, 1000},
			ShadowDepthBias:  DefaultDirectionalLight.ShadowDepthBias,
			ShadowNormalBias: DefaultDirectionalLight.ShadowNormalBias,
		},
	)

	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(-2, 2.5, 5).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)
}
