package main

import (
	"image"
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
		defer profile.Start(profile.MemProfile).Stop()
	}

	app.AddPlugin(PluginRender)
	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.MustRun()
}

func setupSystem(commands *Commands, ctx *RenderContext, assets *Assets) {
	model := assets.GLTF("City.glb").Await()

	skybox := loadSkybox(ctx, assets)

	commands.Spawn(
		Camera{},
		Camera3d,
		HDR{},
		FirstPersonViewController{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(-3.8791254, 2.5908828, 7.1305904),
		Skybox{Texture: skybox},
	)

	commands.Spawn(
		NewTransform().
			// WithScaleXYZ(0.05, 0.05, 0.05).
			WithRotationY(glm.DegToRad(120)),

		SceneRoot{Handle: model},
	)

	commands.Spawn(
		TransformFromXYZ(-3.8791254, 3.0, 7.13),
		PointLight{
			Color:        ColorLinearRGB(100, 100, 100),
			AttQuadratic: 1,
		},
	)
	commands.Spawn(
		TransformFromXYZ(2.6167593, 2.3005552, -4.5687613),
		PointLight{
			Color:        ColorLinearRGB(5, 5, 5),
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

func loadSkybox(ctx *RenderContext, a *Assets) *Texture {
	// layer 0 => positive x
	// layer 1 => negative x
	// layer 2 => positive y
	// layer 3 => negative y
	// layer 4 => positive z
	// layer 5 => negative z

	lf := a.Image("skybox/20250717_211408_0779_lf.png")
	rt := a.Image("skybox/20250717_211408_0779_rt.png")
	up := a.Image("skybox/20250717_211408_0779_up.png")
	dn := a.Image("skybox/20250717_211408_0779_dn.png")
	bk := a.Image("skybox/20250717_211408_0779_bk.png")
	ft := a.Image("skybox/20250717_211408_0779_ft.png")

	// wait for all images to load
	images := []image.Image{
		ft.Await(),
		bk.Await(),
		up.Await(),
		dn.Await(),
		rt.Await(),
		lf.Await(),
	}

	return NewTextureFromImages(ctx, images, TextureFromImagesOptions{
		Label:         "Skybox",
		SRGB:          true,
		ViewDimension: wgpu.TextureViewDimensionCube,
	})
}
