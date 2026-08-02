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
	app.AddSystems(Update, animateMaterialsSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, []int{10, 100, 200})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera3d,
		Camera{},
		// HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(3.0, 1.0, 3.0).LookingAt(glm.Vec3f{0.0, 0.25, 0.0}, glm.Vec3f{0, 1, 0}),
	)

	commands.Spawn(
		TransformFromXYZ(5, 5, 10).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
		PointLight{Color: ColorLinearRGB(1, 1, 1), Intensity: 100},
	)

	cube := CubeWithSize(glm.Vec3f{0.5, 0.5, 0.5})

	color := Oklch{L: 1, C: 0.75, H: 0}

	for x := -1; x < 2; x++ {
		for z := -1; z < 2; z++ {
			commands.Spawn(
				TransformFromXYZ(float32(x), 0, float32(z)),
				MeshInstance{Mesh: cube},
				StandardMaterial{BaseColor: color.ToColor(), PerceptualRoughness: 0.8},
				OkColor{Color: color},
			)

			color.H += float32(glm.DegToRad(137.50777))
		}
	}
}

type OkColor struct {
	Component[OkColor]
	Color Oklch
}

func animateMaterialsSystem(
	vt VirtualTime,
	blocksQuery Query[struct {
		OkColor  *OkColor
		Material *StandardMaterial
	}],
) {
	for item := range blocksQuery.Items() {
		item.OkColor.Color.H += 2 * vt.DeltaSecs
		item.Material.BaseColor = item.OkColor.Color.ToColor()
	}
}
