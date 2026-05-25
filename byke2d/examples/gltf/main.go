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

func setupSystem(world *World, commands *Commands, assets *Assets) {
	model := assets.GLTF("HouseConstructionSite.glb").Await()

	commands.Spawn(
		Camera{},
		TransformFromXYZ(0, -20, 100),
		DefaultPerspectiveProjection,
	)

	commands.Spawn(
		NewTransform().WithRotationY(glm.DegToRad(0)),
		SceneRoot(world, model, 0),
	)
}

func moveCameraSystem(vt VirtualTime, cam Single[struct {
	_         With[Camera]
	Transform *Transform
}]) {
	y := math.Sin(vt.Elapsed.Seconds())*20 - 30
	cam.Get().Transform.Translation[1] = float32(y)
}
