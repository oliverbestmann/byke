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
	app.AddSystems(byke.Update, animateSprite)

	app.MustRun()
}

type Rocket struct {
	byke.ImmutableComponent[Rocket]
}

type Animate struct {
	byke.Component[Animate]
	Timer byke.Timer
}

func setupSystem(commands *byke.Commands, assets *Assets) {
	asset := assets.Texture("circle.png").Await()
	figure := assets.Texture("figure.png").Await()

	commands.Spawn(
		Camera{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1000},
			Scale:          1.0,
		},
	)

	for range 100 {
		x := rand.Float32()*1000 - 500
		y := rand.Float32()*600 - 300
		size := rand.Float32()*32 + 16
		alpha := rand.Float32()*0.8 + 0.1

		commands.Spawn(
			TransformFromXY(x, y),
			Sprite{
				Texture:    asset,
				Color:      wx.ColorSRGBA(1, 1, 1, alpha),
				CustomSize: Some(glm.Vec2f{size, size}),
			},
		)
	}

	gridOptions := GridOptions{
		Count:  12,
		Width:  32,
		Height: 32,
	}

	commands.Spawn(
		Rocket{},
		Sprite{Texture: figure},
		TransformFromXYZ(0, 0, 1).WithScaleXY(4, 4),
		TextureAtlasFromGrid(gridOptions),
		Animate{Timer: byke.NewTimerWithFrequency(4.0)},
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

func animateSprite(vt *byke.VirtualTime, query byke.Query[struct {
	Animation    *Animate
	TextureAtlas *TextureAtlas
}]) {
	for item := range query.Items() {
		if item.Animation.Timer.Tick(vt.Delta).JustFinished() {
			item.TextureAtlas.Index += 1
		}
	}
}
