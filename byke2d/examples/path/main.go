package main

import (
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
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

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.AddSystems(Startup, setupSystem)
	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		Camera2d,
		TransformFromXYZ(0, 0, -0.5),
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 512},
			// ScalingMode: ScalingModeWindowSize{},
		},
	)

	var path Path
	path.MoveTo(glm.Vec2f{-32, -32})
	path.LineTo(glm.Vec2f{-32, 32})
	path.LineTo(glm.Vec2f{16, 32})
	path.QuadCurveTo(glm.Vec2f{32, 32}, glm.Vec2f{32, 16})
	path.LineTo(glm.Vec2f{32, -32})
	path.LineTo(glm.Vec2f{32, -32})
	path.Close()

	commands.Spawn(
		TransformFromXYZ(0, 0, 0.1),
		MeshInstance{Mesh: path.ToMesh(0.1)},
		ColorMaterial{
			Tint: ColorSRGBA(1.0, 0.0, 0.5, 1.0),
		},
	)
}
