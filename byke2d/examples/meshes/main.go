package main

import (
	"embed"
	"log/slog"
	"math/rand/v2"
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

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
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
			Scale: 2.0,
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
		RegularPolygon(32, 3),
	)

	circle := Circle(32, 8)

	// add random colors
	for range circle.Vertices {
		color := ColorSRGBA(rand.Float32(), rand.Float32(), rand.Float32(), 1)
		circle.Colors = append(circle.Colors, color)
	}

	commands.Spawn(
		TransformFromXY(32, 0),
		circle,
	)
}

func rotateSpritesSystem(vt VirtualTime, query Query[struct {
	_         Or[With[Sprite], With[Mesh2d]]
	Transform *Transform
}]) {
	for item := range query.Items() {
		rot := glm.RotationZQuat(glm.Rad(3 * vt.DeltaSecs))
		item.Transform.Rotation = item.Transform.Rotation.Mul(rot)
	}
}
