package main

import (
	"embed"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
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
	app.AddSystems(Update, rotateSpriteSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.MustRun()
}

type SpriteToRotate struct {
	ImmutableComponent[SpriteToRotate]
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

	_ = cameraTexture

	// camera with MSAA activated
	commands.Spawn(
		Camera{},
		MSAA{},
		RenderTarget{Texture: AsRenderTexture(cameraTexture)},
	)

	commands.Spawn(
		TransformFromXY(0, 0).
			WithScaleXY(0.25, 0.25).
			WithRotationZ(1),
		Sprite{Texture: asset},
		SpriteToRotate{},
	)

	// second camera:
	// draw a scaled view of the first camera to show the antialiasing result
	commands.Spawn(
		NewTransform().WithScaleXY(8, 8),
		Camera{Order: 1},
		RenderLayersOf(1),
	)

	commands.Spawn(
		Sprite{Texture: cameraTexture},
		RenderLayersOf(1),
	)
}

func rotateSpriteSystem(vt VirtualTime, sprites Query[struct {
	_         With[SpriteToRotate]
	Transform *Transform
}]) {
	for sprite := range sprites.Items() {
		r := glm.RotationZQuat(glm.Rad(vt.DeltaSecs) * 0.01)
		sprite.Transform.Rotation = sprite.Transform.Rotation.Mul(r)
	}
}
