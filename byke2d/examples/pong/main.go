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
	"github.com/oliverbestmann/webgpu/wgpu"
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
	app.AddSystems(byke.Update, moveSprites)
	app.AddSystems(byke.Update, animateSprite)

	app.MustRun()
}

type Rocket struct {
	byke.ImmutableComponent[Rocket]
}

type Animate struct {
	byke.Component[Animate]
	byke.Timer
}

type Velocity struct {
	byke.Component[Velocity]
	glm.Vec2f
}

func setupSystem(commands *byke.Commands, assets *Assets) {
	asset := assets.Texture("circle.png").Await()

	nnSettings := &LoadTextureSettings{
		Sampler: SamplerConfig{
			FilterMode: wgpu.FilterModeNearest,
		},
	}

	figure := assets.TextureWithSettings("figure.png", nnSettings).Await()

	commands.Spawn(
		Camera{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1000},
			Scale:          1.0,
		},
	)

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

	for range 500 {
		x := (rand.Float32() - 0.5) * 1200
		y := (rand.Float32() - 0.5) * 1200

		vx := (rand.Float32() - 0.5) * 5
		vy := (rand.Float32() - 0.5) * 5

		size := rand.Float32()*32 + 16
		alpha := rand.Float32()*0.8 + 0.1

		commands.Spawn(
			TransformFromXY(x, y),
			Velocity{Vec2f: glm.Vec2f{vx, vy}},
			Sprite{
				Texture:    asset,
				Color:      wx.ColorSRGBA(1, 1, 1, alpha),
				CustomSize: Some(glm.Vec2f{size, size}),
			},
		)
	}
}

func moveSprites(vt byke.VirtualTime, query byke.Query[struct {
	_         byke.With[Sprite]
	Transform *Transform
	Velocity  Velocity
}]) {
	for item := range query.Items() {
		vel := item.Velocity.Scale(vt.DeltaSecs)
		newValue := item.Transform.Translation.Truncate().Add(vel)
		item.Transform.Translation[0] = newValue[0]
		item.Transform.Translation[1] = newValue[1]
	}
}

func animateSprite(vt *byke.VirtualTime, query byke.Query[struct {
	Animation    *Animate
	TextureAtlas *TextureAtlas
}]) {
	for item := range query.Items() {
		if item.Animation.Tick(vt.Delta).JustFinished() {
			item.TextureAtlas.Index += 1
		}
	}
}
