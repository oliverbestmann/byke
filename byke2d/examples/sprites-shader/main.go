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

//go:embed assets/noise.wgsl
var noisyShaderCode string

//go:embed assets/fade.wgsl
var fadeShaderCode string

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

var noisyShader = &ShaderDef{
	Source:        noisyShaderCode,
	FragmentEntry: "noisy_sprite",
}

var fadeShader = &ShaderDef{
	Source:        fadeShaderCode,
	FragmentEntry: "fade_sprite",
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{Order: 1},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixed{Viewport: glm.Vec2f{640, 360}},
			// ScalingMode: ScalingModeWindowSize{},
			Scale: 2.0,
		},
	)

	asset := assets.Texture("marker.png").Await()
	commands.Spawn(
		TransformFromXY(0, 0),
		Sprite{Texture: asset},
		TextureAtlas{Layout: TextureAtlasLayoutFromRect(glm.RectFromXYWH[uint32](0, 0, 4, 32))},
	)

	commands.Spawn(
		TransformFromXY(-32, 0),
		Sprite{Texture: asset},
		AnchorTopLeft,
		CustomShader{Shader: noisyShader},
	)

	commands.Spawn(
		TransformFromXY(32, 0),
		Sprite{Texture: asset},
		AnchorTopRight,
		CustomShader{Shader: fadeShader},
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
