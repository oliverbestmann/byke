package main

import (
	"embed"
	"log/slog"
	"math/rand/v2"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
	"github.com/oliverbestmann/webgpu/wgpu"
	"github.com/pkg/profile"
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

	if runtime.GOOS != "js" {
		defer profile.Start(profile.MemProfileRate(256)).Stop()
	}

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		TransformFromXYZ(0, 0, 0.5).WithScaleXY(2, 2),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 640},
			// ScalingMode: ScalingModeWindowSize{},
		},
	)

	asset := assets.Texture("mazipan.jpg").Await()

	commands.Spawn(
		TransformFromXYZ(0, 0, 0),
		Sprite{Texture: asset},
		TextureAtlas{Layout: TextureAtlasLayoutFromRect(glm.RectuFromXYWH(0, 0, 16, 32))},
	)

	commands.Spawn(
		TransformFromXYZ(-24, 0, -0.1),
		Mesh2d{Mesh: RegularPolygon(32, 3)},
		ColorMaterial{
			Tint:    ColorSRGBA(1.0, 0.0, 0.5, 1.0),
			Texture: asset,
		},
	)

	circle := Circle(32, 64)

	// add random colors
	var colors []Color
	for range circle.Vertices() {
		color := ColorSRGBA(rand.Float32(), rand.Float32(), rand.Float32(), 1)
		colors = append(colors, color)
	}

	circle.WithAttributes(VertexAttributeColor, wgpu.ToBytes(colors))

	for i := 0; i < 3; i++ {
		// circle should be batched
		commands.Spawn(
			TransformFromXYZ(24, float32(-32*i), -0.1).WithScaleXY(0.5, 0.5),
			Mesh2d{Mesh: circle},
			ColorMaterial{},
		)
	}
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
