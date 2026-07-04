package main

import (
	"embed"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
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

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
	)

	asset := assets.Texture("input.jpg").Await()

	commands.Spawn(
		TransformFromXY(-1024, 0).WithScaleXY(2, 2),
		Sprite{Texture: asset},
	)
	commands.Spawn(
		TransformFromXY(-256, 0),
		Sprite{Texture: asset},
	)

	commands.Spawn(
		TransformFromXY(128, 0).WithScaleXY(0.5, 0.5),
		Sprite{Texture: asset},
	)

	commands.Spawn(
		TransformFromXY(128+128+64, 0).WithScaleXY(0.25, 0.25),
		Sprite{Texture: asset},
	)

	commands.Spawn(
		TransformFromXY(128+128+64+64+32, 0).WithScaleXY(0.125, 0.125),
		Sprite{Texture: asset},
	)
}
