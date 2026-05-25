package main

import (
	"embed"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
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

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.AddSystems(Startup, setupSystem)
	app.MustRun()
}

func setupSystem(world *World, commands *Commands, assets *Assets) {
	monkey := assets.GLTF("Monkey.glb").Await()

	commands.Spawn(
		Camera{},
		TransformFromXYZ(0, 0, -1.0),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 16},
			// ScalingMode: ScalingModeWindowSize{},
			Scale: 1.0,
		},
	)

	commands.Spawn(
		NewTransform().WithScaleXYZ(1, 1, 0),
		SceneRoot(world, monkey, 0),
	)
}
