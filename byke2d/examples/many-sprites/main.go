package main

import (
	"log/slog"
	"math/rand/v2"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/pkg/profile"
)

const SpriteCount = 100_000

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))
}

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(FixedUpdate, moveSpritesSystem)

	app.AddSystems(Last, func(vt VirtualTime, w *MessageWriter[AppExit]) {
		if vt.Elapsed.Seconds() > 10 {
			w.Write(AppExitSuccess)
		}
	})

	if runtime.GOOS != "js" {
		defer profile.Start(profile.CPUProfile).Stop()
	}

	app.MustRun()
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

	for idx := range SpriteCount {
		z := float32(idx) / SpriteCount

		commands.Spawn(
			TransformFromXYZ(rand.Float32()*1000, rand.Float32()*600, z).
				WithRotationZ(glm.Rad(rand.Float32())).
				WithScaleXY(rand.Float32()+0.1, rand.Float32()+0.1),

			Sprite{
				Texture:    asset,
				CustomSize: Some(glm.Vec2f{32, 32}),
				Color:      ColorSRGBA(1, 1, 1, 0.01),
			},

			Velocity{Vec2f: glm.Vec2f{rand.Float32() - 0.5, rand.Float32() - 0.5}.Scale(32)},
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
