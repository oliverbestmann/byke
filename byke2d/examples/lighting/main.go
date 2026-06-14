package main

import (
	_ "image/png"
	"log/slog"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
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
	app.AddSystems(Update, moveCameraSystem)
	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	model := assets.GLTF("Fox.glb").Await()

	commands.Spawn(
		Camera{},
		HDR{},
		CameraController{
			Pitch:   glm.DegToRad(10),
			Yaw:     glm.DegToRad(180),
			PosRoll: glm.DegToRad(180),
			PosY:    100,
			Radius:  100,
		},
		DefaultPerspectiveProjection,
	)

	commands.Spawn(
		NewTransform().
			WithRotationY(glm.DegToRad(120)),

		SceneRoot{Handle: model},
	)

	commands.Spawn(
		TransformFromXYZ(-5, 3, 10),
		PointLight{
			Color:        glm.Vec3f{1, 1, 1},
			Intensity:    100,
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

func moveCameraSystem(vt VirtualTime, keys Keys, cam Single[struct {
	_                With[Camera]
	Transform        *Transform
	CameraController *CameraController
}]) {
	c := cam.Get()

	if keys.IsPressed(vyn.KeyE) {
		c.CameraController.PosY += vt.DeltaSecs
	}

	if keys.IsPressed(vyn.KeyQ) {
		c.CameraController.PosY -= vt.DeltaSecs
	}

	if keys.IsPressed(vyn.KeyA) {
		c.CameraController.PosRoll -= glm.Rad(-1 * vt.DeltaSecs)
	}

	if keys.IsPressed(vyn.KeyD) {
		c.CameraController.PosRoll += glm.Rad(-1 * vt.DeltaSecs)
	}

	pos := glm.RotationYQuat(c.CameraController.PosRoll).Transform(glm.Vec3f{0, c.CameraController.PosY, -c.CameraController.Radius})
	c.Transform.Translation = pos

	if keys.IsPressed(vyn.KeyArrowUp) {
		c.CameraController.Pitch += glm.Rad(-1 * vt.DeltaSecs)
	}

	if keys.IsPressed(vyn.KeyArrowDown) {
		c.CameraController.Pitch -= glm.Rad(-1 * vt.DeltaSecs)
	}

	if keys.IsPressed(vyn.KeyArrowLeft) {
		c.CameraController.Yaw += glm.Rad(-1 * vt.DeltaSecs)
	}

	if keys.IsPressed(vyn.KeyArrowRight) {
		c.CameraController.Yaw -= glm.Rad(-1 * vt.DeltaSecs)
	}

	c.Transform.Rotation = glm.RotationXQuat(c.CameraController.Pitch).Mul(glm.RotationYQuat(c.CameraController.Yaw))
}

type CameraController struct {
	Component[CameraController]
	Pitch glm.Rad
	Yaw   glm.Rad

	PosRoll glm.Rad
	PosY    float32
	Radius  float32
}
