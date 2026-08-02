package main

import (
	_ "image/png"
	"math"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Define a component to designate a rotation speed to an entity.
type Rotatable struct {
	Component[Rotatable]
	speed float32
}

var _ = ValidateComponent[Rotatable]()

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, rotateCubeSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 20, 30})
}

func setupSystem(commands *Commands) {
	// Spawn a cube to rotate.
	commands.Spawn(
		MeshInstance{Mesh: Cube()},
		StandardMaterial{BaseColor: ColorSRGB(1, 1, 1), PerceptualRoughness: 0.8},
		Rotatable{speed: 0.3},
	)

	// Spawn a camera looking at the entities to show what's happening in this example.
	commands.Spawn(
		Camera3d,
		Camera{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(0, 5, 10).LookingAt(glm.Vec3f{0, 0, 0}, glm.Vec3f{0, 1, 0}),
	)

	// Add a light source so we can see clearly.
	commands.Spawn(
		PointLight{Color: ColorLinearRGB(1, 1, 1), Intensity: 10000},
		TransformFromXYZ(3, 3, 3).LookingAt(glm.Vec3f{0, 0, 0}, glm.Vec3f{0, 1, 0}),
	)
}

// This system will rotate any entity in the scene with a Rotatable component around its y-axis.
func rotateCubeSystem(
	vt VirtualTime,
	items Query[struct {
		Transform *Transform
		Rotatable Rotatable
	}],
) {
	for item := range items.Items() {
		const TAU = math.Pi * 2

		// The speed is first multiplied by TAU which is a full rotation (360deg) in radians,
		// and then multiplied by delta_secs which is the time that passed last frame.
		// In other words. Speed is equal to the amount of rotations per second.
		*item.Transform = item.Transform.RotateY(glm.Rad(item.Rotatable.speed * TAU * vt.DeltaSecs))
	}
}
