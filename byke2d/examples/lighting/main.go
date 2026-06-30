package main

import (
	_ "image/png"
	"log/slog"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/pkg/profile"
)

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))
}

func main() {
	var app App

	assets := os.DirFS(".")

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	if runtime.GOOS != "js" {
		defer profile.Start(profile.CPUProfile).Stop()
	}

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	model := assets.GLTF("City.glb").Await()

	commands.Spawn(
		Camera{},
		HDR{},
		FirstPersonViewController{},
		DefaultPerspectiveProjection,
	)

	commands.Spawn(
		NewTransform().
			// WithScaleXYZ(0.05, 0.05, 0.05).
			WithRotationY(glm.DegToRad(120)),

		SceneRoot{Handle: model},
	)

	commands.Spawn(
		TransformFromXYZ(-5, 3, 10),
		PointLight{
			Color:        ColorLinearRGB(100, 100, 100),
			AttQuadratic: 1,
		},
	)

	// commands.Spawn(
	// 	TransformFromXYZ(-4, 7, -6),
	// 	PointLight{
	// 		Color:        glm.Vec3f{1, 1, 1},
	// 		Intensity:    2,
	// 		AttQuadratic: 1,
	// 	},
	// )
}
