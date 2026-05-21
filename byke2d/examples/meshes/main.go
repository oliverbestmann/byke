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
	"github.com/oliverbestmann/webgpu/wgpu"
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
	app.AddSystems(Update, System(rotateSystem).RunIf(KeyIsPressed(vyn.KeyR)))

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		TransformFromXYZ(0, 0, -0.5),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixed{Viewport: glm.Vec2f{640, 360}},
			// ScalingMode: ScalingModeWindowSize{},
			Scale: 2.0,
		},
	)

	asset := assets.Texture("marker.png").Await()

	commands.Spawn(
		TransformFromXYZ(0, 0, 0),
		Sprite{Texture: asset},
		TextureAtlas{Layout: TextureAtlasLayoutFromRect(glm.RectuFromXYWH(0, 0, 4, 32))},
	)

	commands.Spawn(
		TransformFromXYZ(-24, 0, 0.1),
		Mesh2d{Mesh: RegularPolygon(32, 3)},
		MeshColor{Color: ColorSRGBA(1.0, 0.0, 0.5, 1.0)},
	)

	circle := Circle(32, 8)

	// add random colors
	var colors []Color
	for range circle.Vertices() {
		color := ColorSRGBA(rand.Float32(), rand.Float32(), rand.Float32(), 1)
		colors = append(colors, color)
	}

	circle.WithAttributes(VertexAttributeColor, wgpu.ToBytes(colors))

	commands.Spawn(
		TransformFromXYZ(24, 0, -0.1),
		Mesh2d{Mesh: circle},
	)
}

func rotateSystem(vt VirtualTime, query Query[struct {
	_         Or[With[Sprite], With[Mesh2d]]
	Transform *Transform
}]) {
	for item := range query.Items() {
		rot := glm.RotationZQuat(glm.Rad(3 * vt.DeltaSecs))
		item.Transform.Rotation = item.Transform.Rotation.Mul(rot)
	}
}
