package main

import (
	"embed"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
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
	app.AddSystems(byke.Update, moveSprite)

	app.MustRun()
}

func setupSystem(commands *byke.Commands, assets *byke2d.Assets) {
	asset := assets.Texture("circle.png").Await()

	commands.Spawn(
		byke2d.Camera{},
		byke2d.OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    byke2d.ScalingModeWindowSize{},
			Scale:          1.0,
		},
	)

	for range 100 {
		x := rand.Float32()*1000 - 500
		y := rand.Float32()*600 - 300
		tr := byke2d.NewTransform()
		tr.Translation = glm.Vec3f{x, y, 0}

		commands.Spawn(
			tr,
			byke2d.Sprite{Texture: asset},
		)
	}
}

func moveCamera(query byke.Query[struct {
	_         byke.With[byke2d.Camera]
	Transform *byke2d.Transform
}]) {
	for item := range query.Items() {
		item.Transform.Translation[0] += 1
	}
}

func moveSprite(query byke.Query[struct {
	_         byke.With[byke2d.Sprite]
	Transform *byke2d.Transform
}]) {
	for item := range query.Items() {
		item.Transform.Rotation += 0.01
	}
}
