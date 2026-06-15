package main

import (
	"embed"
	"log/slog"
	"math"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
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

	if runtime.GOOS != "js" {
		defer profile.Start(profile.CPUProfile).Stop()
	}

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, moveCameraSystem)
	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	model := assets.GLTF("HouseConstructionSite.glb").Await()

	commands.Spawn(
		Camera{},
		HDR{},
		TransformFromXYZ(0, 20, -100),
		DefaultPerspectiveProjection,
	)

	commands.Spawn(
		NewTransform().WithRotationY(glm.DegToRad(0)),
		SceneRoot{Handle: model},
	)

	commands.Spawn(
		TransformFromXYZ(20, 20, -50),
		PointLight{
			Color:        ColorLinearRGB(glm.Vec3f{1, 1, 1}.Scale(20).XYZ()),
			AttQuadratic: 0.1,
		},
	)

	commands.Spawn(
		NewTransform().WithRotationX(0.8),
		DirectionalLight{Color: ColorLinearRGB(0.2, 0.1, 0.0)},
	)
}

func moveCameraSystem(vt VirtualTime, cam Single[struct {
	_         With[Camera]
	Transform *Transform
}]) {
	y := math.Sin(vt.Elapsed.Seconds())*20 + 30
	cam.Get().Transform.Translation[1] = float32(y)
}
