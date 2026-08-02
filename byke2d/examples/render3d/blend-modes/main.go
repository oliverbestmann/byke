package main

import (
	_ "image/png"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Startup, setupFloorTilesSystem)
	app.AddSystems(Update, rotateCameraSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(8.0, 2.5, 1.0).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)

	commands.Spawn(
		TransformFromXYZ(4, 8, 4),
		PointLight{
			Color:     ColorLinearRGB(1, 1, 1),
			Intensity: 100000,
		},
	)

	sphere := Sphere(0.9, 32, 18)

	baseColor := ColorSRGBA(0.9, 0.2, 0.3, 0.9)

	alphaModes := []AlphaMode{
		AlphaModeOpaque,
		AlphaModeMask,
		AlphaModeBlend,
		Premultiplied,
		AlphaModeAdd,
		AlphaModeMultiply,
	}

	for idx, alphaMode := range alphaModes {
		commands.Spawn(
			TransformFromXYZ(-5+float32(2*idx), 0, 0),
			MeshInstance{Mesh: sphere},
			StandardMaterial{BaseColor: baseColor, AlphaMode: alphaMode, PerceptualRoughness: 0.5},
		)
	}
}

func setupFloorTilesSystem(
	commands *Commands,
) {

	plane := PlaneWithSize(glm.Vec2f{2, 2})

	for x := -4; x < 5; x++ {
		for z := -4; z < 5; z++ {
			var tileColor Color
			if (x+z)%2 == 0 {
				tileColor = ColorSRGB(1, 1, 1)
			} else {
				tileColor = ColorSRGB(0.2, 0.2, 0.2)
			}

			commands.Spawn(
				TransformFromXYZ(float32(2*x), -1, float32(2*z)),
				MeshInstance{Mesh: plane},
				StandardMaterial{BaseColor: tileColor, PerceptualRoughness: 0.8},
			)
		}
	}
}

func rotateCameraSystem(
	vt VirtualTime,
	keys Keys,
	cameraQuery Single[struct {
		_         With[Camera]
		Transform *Transform
	}],
) {
	camera := cameraQuery.Get()

	var rotation glm.Rad
	if keys.IsPressed(vyn.KeyArrowLeft) {
		rotation += glm.Rad(vt.DeltaSecs)
	}

	if keys.IsPressed(vyn.KeyArrowRight) {
		rotation -= glm.Rad(vt.DeltaSecs)
	}

	*camera.Transform = camera.Transform.RotateAround(
		glm.Vec3f{}, glm.RotationYQuat(rotation),
	)
}
