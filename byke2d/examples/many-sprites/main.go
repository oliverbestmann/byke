package main

import (
	"embed"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"runtime"
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/pkg/profile"
)

const SpriteCount = 100_000

//go:embed assets
var assets embed.FS

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))
}

func main() {
	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))
	app.InsertResource(fpsCounter{})

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, updateFpsCounterSystem)
	app.AddSystems(FixedUpdate, moveSpritesSystem)

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

	commands.Spawn(
		AnchorBottomLeft,
		TransformFromXYZ(32, 32, 2),
		Text{Text: "Hello", Size: 16.0, Color: ColorSRGBA(1, 0, 1, 1)},
	)
}

func moveSpritesSystem(t FixedTime, query Query[struct {
	Transform *Transform
	Velocity  Velocity
}]) {
	for item := range query.Items() {
		posNew := item.Transform.Translation.
			Truncate().
			Add(item.Velocity.Scale(t.DeltaSecs))

		item.Transform.Translation[0] = posNew[0]
		item.Transform.Translation[1] = posNew[1]
	}
}

func updateFpsCounterSystem(
	counter *fpsCounter,
	text Single[*Text],
) {
	fps := counter.Update()
	text.Value.Text = fmt.Sprintf("FPS: %1.2f", fps)
}

type fpsCounter struct {
	timestamps []time.Time
}

func (c *fpsCounter) Update() float32 {
	now := time.Now()
	c.timestamps = append(c.timestamps, now)

	if len(c.timestamps) > 120 {
		c.timestamps = c.timestamps[1:]
	}

	return float32(len(c.timestamps)) / float32(now.Sub(c.timestamps[0]).Seconds())
}
