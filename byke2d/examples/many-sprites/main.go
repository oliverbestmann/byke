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
	"github.com/oliverbestmann/pulse/glm"
	"github.com/pkg/profile"
)

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

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, updateFpsCounterSystem)

	if runtime.GOOS != "js" {
		defer profile.Start(profile.CPUProfile).Stop()
	}

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{Order: 0},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.0, 0.0},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1000},
			Scale:          1.0,
		},
	)

	asset := assets.Texture("circle.png").Await()

	for range 100_000 {
		commands.Spawn(
			TransformFromXYZ(rand.Float32()*1000, rand.Float32()*600, rand.Float32()),
			Sprite{
				Texture:    asset,
				CustomSize: Some(glm.Vec2f{32, 32}),
				Color:      ColorSRGBA(1, 1, 1, 0.01),
			},
		)
	}

	commands.Spawn(
		AnchorBottomLeft,
		TransformFromXYZ(32, 32, 2),
		Text{Text: "Hello", Size: 16.0, Color: ColorSRGBA(1, 0, 1, 1)},
	)
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
