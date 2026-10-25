package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/transforms/translation/

var _ = ValidateComponent[Movable]()

// Movable moves an entity back and forth along its local x axis.
type Movable struct {
	Component[Movable]
	Spawn       glm.Vec3f
	MaxDistance float32
	Speed       float32
}

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, moveCubeSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 100, 200})
}

func setupSystem(commands *Commands) {
	// Spawn a cube to move.
	commands.Spawn(
		MeshInstance{Mesh: Cube()},
		StandardMaterial{BaseColor: ColorSRGB(1, 1, 1), PerceptualRoughness: 0.8},
		NewTransform(),
		Movable{MaxDistance: 5, Speed: 2},
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
		TransformFromXYZ(3, 3, 3),
	)
}

// This system moves the cube along its local x axis and turns it around
// when it is too far away from its spawn point.
func moveCubeSystem(
	vt VirtualTime,
	items Query[struct {
		Transform *Transform
		Movable   *Movable
	}],
) {
	for item := range items.Items() {
		if item.Movable.Spawn.Sub(item.Transform.Translation).Length() > item.Movable.MaxDistance {
			item.Movable.Speed *= -1
		}

		direction := item.Transform.Rotation.Transform(glm.Vec3f{1, 0, 0})
		item.Transform.Translation = item.Transform.Translation.Add(direction.Scale(item.Movable.Speed * vt.DeltaSecs))
	}
}
