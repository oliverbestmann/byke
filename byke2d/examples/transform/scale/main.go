package main

import (
	_ "image/png"
	"math"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/transforms/scale/

var _ = ValidateComponent[Scaling]()

const X = 0
const Y = 1
const Z = 2

// Define a component to keep information for the scaled object.
type Scaling struct {
	Component[Scaling]
	scaleDirection glm.Vec3f
	scaleSpeed     float32
	maxElementSize float32
	minElementSize float32
}

// NewScaling is a simple initialization function.
func NewScaling() Scaling {
	return Scaling{
		scaleDirection: glm.Vec3f{X: 1, Y: 0, Z: 0},
		scaleSpeed:     2.0,
		maxElementSize: 5.0,
		minElementSize: 1.0,
	}
}

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, changeScaleDirectionSystem)
	app.AddSystems(Update, scaleCubeSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 100, 200, 300, 400, 600})
}

func setupSystem(commands *Commands) {
	// Spawn a cube to scale.
	commands.Spawn(
		MeshInstance{Mesh: Cube()},
		StandardMaterial{Tint: ColorSRGB(1, 1, 1)},
		TransformFromXYZ(0, 0, 0).RotateY(math.Pi/4),
		NewScaling(),
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
		PointLight{Color: ColorLinearRGB(5, 5, 5), AttQuadratic: 1},
		TransformFromXYZ(3, 3, 3).LookingAt(glm.Vec3f{0, 0, 0}, glm.Vec3f{0, 1, 0}),
	)
}

// This system will check if a scaled entity went above or below the entities scaling bounds
// and change the direction of the scaling vector.
func changeScaleDirectionSystem(
	items Query[struct {
		Transform *Transform
		Scaling   *Scaling
	}],
) {
	for item := range items.Items() {
		// If an entity scaled beyond the maximum of its size in any dimension
		// the scaling vector is flipped so the scaling is gradually reverted.
		// Additionally, to ensure the condition does not trigger again we floor the elements to
		// their next full value, which should be max_element_size at max.
		scale := item.Transform.Scale
		maxElement := scale[X]
		if scale[Y] > maxElement {
			maxElement = scale[Y]
		}
		if scale[Z] > maxElement {
			maxElement = scale[Z]
		}

		if maxElement > item.Scaling.maxElementSize {
			item.Scaling.scaleDirection = item.Scaling.scaleDirection.Scale(-1)
			item.Transform.Scale = glm.Vec3f{
				X: float32(math.Floor(float64(scale[X]))),
				Y: float32(math.Floor(float64(scale[Y]))),
				Z: float32(math.Floor(float64(scale[Z]))),
			}
		}

		// If an entity scaled beyond the minimum of its size in any dimension
		// the scaling vector is also flipped.
		// Additionally the Values are ceiled to be min_element_size at least
		// and the scale direction is flipped.
		// This way the entity will change the dimension in which it is scaled any time it
		// reaches its min_element_size.
		minElement := scale[X]
		if scale[Y] < minElement {
			minElement = scale[Y]
		}
		if scale[Z] < minElement {
			minElement = scale[Z]
		}

		if minElement < item.Scaling.minElementSize {
			item.Scaling.scaleDirection = item.Scaling.scaleDirection.Scale(-1)
			item.Transform.Scale = glm.Vec3f{
				X: float32(math.Ceil(float64(scale[X]))),
				Y: float32(math.Ceil(float64(scale[Y]))),
				Z: float32(math.Ceil(float64(scale[Z]))),
			}
			// Rotate the scale direction (x, y, z) -> (z, x, y)
			item.Scaling.scaleDirection = glm.Vec3f{
				X: item.Scaling.scaleDirection[Z],
				Y: item.Scaling.scaleDirection[X],
				Z: item.Scaling.scaleDirection[Y],
			}
		}
	}
}

// This system will scale any entity with assigned Scaling in each direction
// by cycling through the directions to scale.
func scaleCubeSystem(
	vt VirtualTime,
	items Query[struct {
		Transform *Transform
		Scaling   Scaling
	}],
) {
	for item := range items.Items() {
		scaleFactor := item.Scaling.scaleSpeed * vt.DeltaSecs
		item.Transform.Scale = item.Transform.Scale.Add(
			item.Scaling.scaleDirection.Scale(scaleFactor),
		)
	}
}
