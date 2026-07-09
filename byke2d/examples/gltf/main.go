package main

import (
	"embed"
	"log/slog"
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

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, moveLightsInCircleSystem)
	app.MustRun()
}

type MovingLight struct {
	Component[MovingLight]
	Offset      glm.Vec3f
	AngleOffset glm.Rad
}

func setupSystem(commands *Commands, assets *Assets) {
	sphere := assets.GLTF("Sphere.glb").Await()
	model := assets.GLTF("TestScene.gltf").Await()

	commands.Spawn(
		Camera{},
		HDR{},
		FirstPersonViewController{},
		TransformFromXYZ(5, 5, 5),
		DefaultPerspectiveProjection,
	)

	commands.Spawn(
		SceneRoot{Handle: model},
	)

	commands.Spawn(
		SceneRoot{Handle: sphere},
		MovingLight{Offset: glm.Vec3f{5, 3, 0}},
		PointLight{
			Color:        ColorLinearRGB(glm.Vec3f{1, 1, 0}.Scale(20).XYZ()),
			AttQuadratic: 1,
		},
	)

	commands.Spawn(
		SceneRoot{Handle: sphere},
		MovingLight{Offset: glm.Vec3f{4, 2, 0}, AngleOffset: glm.Rad(1)},
		PointLight{
			Color:        ColorLinearRGB(glm.Vec3f{1, 1, 1}.Scale(5).XYZ()),
			AttQuadratic: 1,
		},
	)

	commands.Spawn(
		SceneRoot{Handle: sphere},
		MovingLight{Offset: glm.Vec3f{3, 8, 0}, AngleOffset: glm.Rad(3.141)},
		PointLight{
			Color:        ColorLinearRGB(glm.Vec3f{0, 0, 1}.Scale(10).XYZ()),
			AttQuadratic: 1,
		},
	)
}

func moveLightsInCircleSystem(
	vt VirtualTime,

	query Query[struct {
		_           With[PointLight]
		MovingLight MovingLight
		Transform   *Transform
	}],
) {
	for item := range query.Items() {
		r := glm.RotationYQuat(glm.Rad(vt.Elapsed.Seconds()) + item.MovingLight.AngleOffset)
		pos := r.Transform(item.MovingLight.Offset)

		item.Transform.Translation = pos
	}
}
