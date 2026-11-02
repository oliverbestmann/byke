package main

import (
	"math/rand/v2"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

const SpriteCount = 100_000

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginFpsCounter)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(FixedUpdate, moveSpritesSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 50})
}

type Velocity struct {
	Component[Velocity]
	glm.Vec2f
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{Order: 0},
		OrthographicProjection{
			ViewportOrigin: glm.V2f(0.0, 0.0),
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1000},
		},
	)

	asset := assets.Texture("circle.png").Await()

	// fixed seed, so every run spawns the same sprites
	rng := rand.New(rand.NewPCG(1, 2))

	rect := Rectangle(glm.V2f(32, 32))

	for idx := range SpriteCount {
		z := float32(idx) / SpriteCount

		commands.Spawn(
			TransformFromXYZ(rng.Float32()*1000, rng.Float32()*600, z).
				WithRotationZ(glm.Rad(rng.Float32())).
				WithScaleXY(rng.Float32()+0.1, rng.Float32()+0.1),

			MeshInstance{Mesh: rect},

			ColorMaterial{
				Texture:     asset,
				AlphaMode:   AlphaModeMask,
				AlphaCutoff: 0.5,
			},

			Velocity{Vec2f: glm.V2f(rng.Float32()-0.5, rng.Float32()-0.5).Scale(32)},
		)
	}
}

func moveSpritesSystem(t FixedTime, query Query[struct {
	Transform *Transform
	Velocity  Velocity
}],
) {
	for item := range query.Items() {
		posNew := item.Transform.Translation.
			Truncate().
			Add(item.Velocity.Scale(t.DeltaSecs))

		item.Transform.Translation.X = posNew.X
		item.Transform.Translation.Y = posNew.Y
	}
}
