package main

import (
	"embed"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/vyn"
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
			Scale: 4.0,
		},
	)

	asset := assets.Texture("marker.png").Await()
	commands.Spawn(
		Sprite{Texture: asset},
		TextureAtlas{Layout: TextureAtlasLayoutFromRect(glm.RectFromXYWH[uint32](0, 0, 4, 32))},
	)

	commands.Spawn(
		Sprite{Texture: asset, Color: ColorSRGBA(1, 0, 0, 0.5)},
		AnchorTopLeft,
	)

	commands.Spawn(
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
		item.Transform.Rotation += glm.Rad(3 * vt.DeltaSecs)
	}
}
