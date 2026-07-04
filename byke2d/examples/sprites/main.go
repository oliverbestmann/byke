package main

import (
	"embed"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
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

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, flipAllSpritesSystem)
	app.AddSystems(Update, System(rotateSpritesSystem).RunIf(KeyIsPressed(vyn.KeyR)))

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{Order: 1},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixed{Viewport: glm.Vec2f{640, 360}},
			// ScalingMode: ScalingModeWindowSize{},
		},
	)

	asset := assets.Texture("marker.png").Await()
	commands.Spawn(
		TransformFromXY(0, 0),
		Sprite{Texture: asset},
		TextureAtlas{Layout: TextureAtlasLayoutFromRect(glm.RectuFromXYWH(0, 0, 4, 32))},
	)

	commands.Spawn(
		TransformFromXY(-32, 0),
		Sprite{Texture: asset, Color: ColorSRGBA(1, 0, 0, 0.5)},
		AnchorTopLeft,
	)

	commands.Spawn(
		TransformFromXY(32, 0),
		Sprite{Texture: asset, Color: ColorSRGBA(0, 1, 0, 0.5)},
		AnchorTopRight,
	)
}

func flipAllSpritesSystem(keys Keys, query Query[struct {
	Sprite *Sprite
}]) {
	for item := range query.Items() {
		if keys.IsJustPressed(vyn.KeyX) {
			item.Sprite.FlipX = !item.Sprite.FlipX
		}

		if keys.IsJustPressed(vyn.KeyY) {
			item.Sprite.FlipY = !item.Sprite.FlipY
		}
	}
}

func rotateSpritesSystem(vt VirtualTime, query Query[struct {
	Sprite    *Sprite
	Transform *Transform
}]) {
	for item := range query.Items() {
		rot := glm.RotationZQuat(glm.Rad(3 * vt.DeltaSecs))
		item.Transform.Rotation = item.Transform.Rotation.Mul(rot)
	}
}
