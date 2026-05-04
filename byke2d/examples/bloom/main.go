package main

import (
	"embed"
	"log/slog"
	"math"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
)

//go:embed assets
var assets embed.FS

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))

	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, scaleColorSystem)

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		BloomNatural,
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			// ScalingMode:    ScalingModeFixed{Viewport: glm.Vec2f{640, 360}},
			ScalingMode: ScalingModeWindowSize{},
			Scale:       4.0,
		},
	)

	asset := assets.Texture("marker.png").Await()
	commands.Spawn(
		Sprite{
			Texture: asset,
			Color:   wx.ColorSRGBA(1, 1, 1, 1),
		},
	)
}

func scaleColorSystem(vt VirtualTime, query Query[struct {
	Sprite *Sprite
}]) {
	for item := range query.Items() {
		c := float32((math.Sin(vt.Elapsed.Seconds())+1.0)*3.0 + 1.0)
		item.Sprite.Color = wx.ColorSRGBA(c, c, c, 1)
	}
}
