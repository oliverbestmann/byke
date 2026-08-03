package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, System(rotateSystem).RunIf(KeyIsPressed(vyn.KeyR)))

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	const viewWidth = 1000

	uvTexture := assets.Texture("uv.png").Await()

	commands.Spawn(
		Camera{},
		Camera2d,
		TransformFromXYZ(0, 0, -0.5),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: viewWidth},
		},
	)

	meshes := []*Mesh{
		Circle(50, 32),
		Ellipse(glm.Vec2f{100, 50}, 32),
		Rectangle(glm.Vec2f{80, 60}),
		Triangle(50),
		RegularPolygon(50, 6),
		CircularSector(50, 1, 16),
	}

	numShapes := len(meshes)
	for idx, mesh := range meshes {
		f := float32(idx) / float32(numShapes)

		c := Oklch{L: 1, C: 0.5, H: glm.DegToRad(360 * f).Float32()}

		x := (float32(idx+1)/float32(numShapes+1) - 0.5) * viewWidth
		commands.Spawn(
			TransformFromXY(x, 100),
			ColorMaterial{Color: c.ToColor()},
			MeshInstance{Mesh: mesh},
		)

		commands.Spawn(
			TransformFromXY(x, -100),
			ColorMaterial{Texture: uvTexture},
			MeshInstance{Mesh: mesh},
		)
	}
}

func rotateSystem(vt VirtualTime, query Query[struct {
	_         With[MeshInstance]
	Transform *Transform
}],
) {
	for item := range query.Items() {
		rot := glm.RotationZQuat(glm.Rad(3 * vt.DeltaSecs))
		item.Transform.Rotation = item.Transform.Rotation.Mul(rot)
	}
}
