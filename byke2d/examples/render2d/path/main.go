package main

import (
	"math"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Builds meshes from paths with lines and curves.
// The shapes are concave, earcut splits them into triangles.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 600},
		},
	)

	shapes := []struct {
		Path  Path
		Color Color
	}{
		{heart(), ColorSRGB(1, 0.2, 0.4)},
		{star(), ColorSRGB(1, 0.8, 0.2)},
		{crescent(), ColorSRGB(0.4, 0.7, 1)},
	}

	for idx, shape := range shapes {
		commands.Spawn(
			TransformFromXY(float32(idx-1)*190, 0),
			MeshInstance{Mesh: shape.Path.ToMesh(0.1)},
			ColorMaterial{Color: shape.Color},
		)
	}
}

func heart() Path {
	var path Path
	path.MoveTo(glm.Vec2f{0, -70})
	path.CubicCurveTo(glm.Vec2f{-20, -50}, glm.Vec2f{-80, -20}, glm.Vec2f{-80, 20})
	path.CubicCurveTo(glm.Vec2f{-80, 60}, glm.Vec2f{-20, 70}, glm.Vec2f{0, 40})
	path.CubicCurveTo(glm.Vec2f{20, 70}, glm.Vec2f{80, 60}, glm.Vec2f{80, 20})
	path.CubicCurveTo(glm.Vec2f{80, -20}, glm.Vec2f{20, -50}, glm.Vec2f{0, -70})
	path.Close()
	return path
}

func star() Path {
	var path Path

	for idx := range 10 {
		radius := float32(80)
		if idx%2 == 1 {
			radius = 35
		}

		angle := math.Pi/2 + float64(idx)*math.Pi/5
		point := glm.Vec2f{float32(math.Cos(angle)), float32(math.Sin(angle))}.Scale(radius)

		if idx == 0 {
			path.MoveTo(point)
		} else {
			path.LineTo(point)
		}
	}

	path.Close()
	return path
}

func crescent() Path {
	var path Path
	path.MoveTo(glm.Vec2f{20, 75})
	path.QuadCurveTo(glm.Vec2f{-140, 0}, glm.Vec2f{20, -75})
	path.QuadCurveTo(glm.Vec2f{-60, 0}, glm.Vec2f{20, 75})
	path.Close()
	return path
}
