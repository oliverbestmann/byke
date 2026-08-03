package main

import (
	_ "image/png"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
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
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	// no compare, just write snapshot for now
	shared.WriteSnapshots = true

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	opts := &LoadTextureSettings{OverrideTextureViewDimension: wgpu.TextureViewDimensionCube}
	specular := assets.TextureWithSettings("pisa_specular_rgb9e5_zstd.ktx2", opts)
	diffuse := assets.TextureWithSettings("pisa_diffuse_rgb9e5_zstd.ktx2", opts)

	model := assets.GLTF("City.glb").Await()

	skybox := loadSkybox(assets)

	commands.InsertResource(GlobalAmbientLightNone)

	commands.Spawn(
		Camera{},
		Camera3d,

		// HDR{},
		// MSAA{},

		FirstPersonViewController{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(-3.8791254, 2.5908828, 7.1305904),
		Skybox{Texture: skybox},

		EnvironmentMapLight{
			DiffuseMap:  diffuse.Await(),
			SpecularMap: specular.Await(),
			Intensity:   900,
		},
	)

	commands.Spawn(
		NewTransform().
			// WithScaleXYZ(0.05, 0.05, 0.05).
			WithRotationY(glm.DegToRad(120)),

		SceneRoot{
			Handle:                model,
			PreferAlphaToCoverage: false,
		},
	)
}

func loadSkybox(a *Assets) *Texture {
	opts := LoadTextureSettings{OverrideTextureViewDimension: wgpu.TextureViewDimensionCube}
	return a.TextureWithSettings("skybox/pisa.ktx2", &opts).Await()
}
