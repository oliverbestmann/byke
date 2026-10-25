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
			ViewportOrigin: glm.Vec2f{0.0, 0.0},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1000},
		},
	)

	asset := assets.Texture("circle.png").Await()

	// fixed seed, so every run spawns the same sprites
	rng := rand.New(rand.NewPCG(1, 2))

	for idx := range SpriteCount {
		z := float32(idx) / SpriteCount

		commands.Spawn(
			TransformFromXYZ(rng.Float32()*1000, rng.Float32()*600, z).
				WithRotationZ(glm.Rad(rng.Float32())).
				WithScaleXY(rng.Float32()+0.1, rng.Float32()+0.1),

			Sprite{
				Texture:    asset,
				CustomSize: Some(glm.Vec2f{32, 32}),
				Color:      ColorSRGBA(1, 1, 1, 0.01),
			},

			Velocity{Vec2f: glm.Vec2f{rng.Float32() - 0.5, rng.Float32() - 0.5}.Scale(32)},
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

		item.Transform.Translation[0] = posNew[0]
		item.Transform.Translation[1] = posNew[1]
	}
}
