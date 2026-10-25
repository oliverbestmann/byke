package main

import (
	"fmt"
	"math/rand/v2"
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

const SpriteCount = 100_000

func main() {
	var app App

	app.InsertResource(fpsCounter{})

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, updateFpsCounterSystem)
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

	rect := Rectangle(glm.Vec2f{32, 32})

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

			Velocity{Vec2f: glm.Vec2f{rng.Float32() - 0.5, rng.Float32() - 0.5}.Scale(32)},
		)
	}

	commands.Spawn(
		AnchorBottomLeft,
		TransformFromXYZ(32, 32, 1),
		Text{Text: "Hello", Size: 16.0, Color: ColorSRGBA(1, 0, 1, 1)},
	)
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

func updateFpsCounterSystem(
	vt VirtualTime,
	counter *fpsCounter,
	text Single[*Text],
) {
	fps := counter.Update(vt.Elapsed)
	text.Value.Text = fmt.Sprintf("FPS: %1.2f", fps)
}

type fpsCounter struct {
	// virtual time, so the text is the same in every test run
	timestamps []time.Duration
}

func (c *fpsCounter) Update(now time.Duration) float32 {
	c.timestamps = append(c.timestamps, now)

	if len(c.timestamps) > 120 {
		c.timestamps = c.timestamps[1:]
	}

	return float32(len(c.timestamps)) / float32((now - c.timestamps[0]).Seconds())
}
