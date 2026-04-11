package main

import (
	"embed"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"

	"github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/vyn"
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

	var app byke.App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(Plugin)
	app.AddSystems(byke.Update, ExitOnEscapeSystem)

	app.AddSystems(byke.Update, func(buttons MouseButtons, cursor MouseCursor) {
		if buttons.IsJustPressed(vyn.MouseButton(0)) {
			fmt.Println(cursor.XY())
		}
	})

	app.AddSystems(byke.Startup, setupSystem)
	app.AddSystems(byke.Update, moveSprite)

	app.MustRun()
}

type Rocket struct {
	byke.ImmutableComponent[Rocket]
}

func setupSystem(commands *byke.Commands, assets *Assets) {
	asset := assets.Texture("circle.png").Await()
	figure := assets.Texture("figure.png").Await()

	commands.Spawn(
		Camera{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeWindowSize{},
			Scale:          1.0,
		},
	)

	for range 100 {
		x := rand.Float32()*1000 - 500
		y := rand.Float32()*600 - 300

		tr := TransformFromXY(x, y)

		commands.Spawn(
			tr,
			Sprite{Texture: asset},
		)
	}

	commands.Spawn(
		Rocket{},
		Sprite{Texture: figure},
		TransformFromXYZ(0, 0, 1),
		TextureAtlasFromRect(wx.RectangleFromXYWH[uint32](0, 0, 32, 32)),
	)
}

func moveCamera(query byke.Query[struct {
	_         byke.With[Camera]
	Transform *Transform
}]) {
	for item := range query.Items() {
		item.Transform.Translation[0] += 1
	}
}

func moveSprite(query byke.Query[struct {
	_         byke.With[Sprite]
	Transform *Transform
}]) {
	for item := range query.Items() {
		item.Transform.Translation[0] *= 1.001
	}
}
