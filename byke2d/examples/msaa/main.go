package main

import (
	"embed"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
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

	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)
	// app.AddSystems(Update, rotateSystem)

	app.MustRun()
}

func setupSystem(commands *Commands, ctx *RenderContext, assets *Assets) {
	asset := assets.Texture("input.jpg").Await()

	cameraTexture := NewTexture(ctx, NewTextureOptions{
		SamplerConfig: SamplerConfig{
			FilterMode: wgpu.FilterModeNearest,
		},

		Format:       wgpu.TextureFormatBGRA8UnormSrgb,
		Width:        500,
		Height:       300,
		MipmapLevels: 1,
	})

	commands.Spawn(
		Camera{},
		MsaaOn,
		RenderTarget{Texture: AsRenderTexture(cameraTexture)},
	)

	commands.Spawn(
		TransformFromXY(0, 0).WithScaleXY(0.5, 0.5).WithRotation(1),
		Sprite{Texture: asset},
	)

	commands.Spawn(
		NewTransform().WithScaleXY(4, 4),
		Camera{Order: 1},
		MsaaOn,
		RenderLayersOf(1),
	)

	commands.Spawn(
		Sprite{Texture: cameraTexture},
		RenderLayersOf(1),
	)
}

func rotateSystem(vt VirtualTime, query Query[struct {
	_         With[Sprite]
	Transform *Transform
}]) {
	for item := range query.Items() {
		item.Transform.Rotation += glm.Rad(vt.DeltaSecs * 0.1)
	}
}
