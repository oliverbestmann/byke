package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/vyn"
)

//go:embed assets
var assets embed.FS

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))

	var app byke.App

	// configure assets before loading the plugin
	app.InsertResource(byke2d.MakeAssetFS(assets))

	app.AddPlugin(byke2d.Plugin)
	app.AddSystems(byke.Update, byke2d.ExitOnEscapeSystem)

	app.AddSystems(byke.Update, func(buttons byke2d.MouseButtons, cursor byke2d.MouseCursor) {
		if buttons.IsJustPressed(vyn.MouseButton(0)) {
			fmt.Println(cursor.XY())
		}
	})

	app.AddSystems(byke.Startup, setupSystem)

	app.MustRun()
}

func setupSystem(commands *byke.Commands, assets *byke2d.Assets) {
	asset := assets.Texture("circle.png").Await()

	commands.Spawn(
		byke2d.Sprite{
			Texture: asset,
		},
	)
}
