package main

import (
	_ "image/png"
	"log/slog"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
	"github.com/pkg/profile"
)

// //go:embed assets
// var assets embed.FS
var assets = os.DirFS(".")

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
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.MustRun()
}

func setupSystem(commands *Commands, ctx *RenderContext, assets *Assets) {
	opts := &LoadTextureSettings{OverrideTextureViewDimension: wgpu.TextureViewDimensionCube}
	specular := assets.TextureWithSettings("pisa_specular_rgb9e5_zstd.ktx2", opts)
	diffuse := assets.TextureWithSettings("pisa_diffuse_rgb9e5_zstd.ktx2", opts)

	model := assets.GLTF("City.glb").Await()

	skybox := loadSkybox(assets)

	commands.Spawn(
		Camera{},
		Camera3d,
		HDR{},
		FirstPersonViewController{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(-3.8791254, 2.5908828, 7.1305904),
		Skybox{Texture: skybox},

		EnvironmentMapLight{
			DiffuseMap:  diffuse.Await(),
			SpecularMap: specular.Await(),
			Intensity:   1,
		},
	)

	commands.Spawn(
		NewTransform().
			// WithScaleXYZ(0.05, 0.05, 0.05).
			WithRotationY(glm.DegToRad(120)),

		SceneRoot{
			Handle:                model,
			PreferAlphaToCoverage: true,
		},
	)
}

func loadSkybox(a *Assets) *Texture {
	opts := LoadTextureSettings{OverrideTextureViewDimension: wgpu.TextureViewDimensionCube}
	return a.TextureWithSettings("skybox/pisa.ktx2", &opts).Await()
}
