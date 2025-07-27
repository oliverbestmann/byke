package main

import (
	"embed"
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
)

//go:embed assets
var assets embed.FS

func main() {
	var app App

	app.InsertResource(MakeAssetFS(assets))

	// Add the bykebiten game plugin
	app.AddPlugin(GamePlugin)

	// Add a WindowConfig to set title and initial size
	app.InsertResource(WindowConfig{
		Title:  "AssetDucky",
		Width:  800,
		Height: 600,
	})

	// configure plugins
	app.AddPlugin(pluginScreen)
	app.AddPlugin(pluginPause)
	app.AddPlugin(pluginMenu)
	app.AddPlugin(pluginGame)

	// preload assets
	assets, _ := ResourceOf[Assets](app.World())
	assets.Image("ebiten.png")
	assets.Image("ducky.png")

	// spawn the camera
	app.AddSystems(Startup, spawnCameraSystem)

	fmt.Println(app.Run())
}

func spawnCameraSystem(commands *Commands) {

	commands.Spawn(
		Camera{
			ClearColor: &color.Color{
				R: 0.1,
				G: 0.1,
				B: 0.1,
				A: 1.0,
			},

			// SubCameraView: &gm.Rect{
			// 	Min: gm.VecSplat(0.2),
			// 	Max: gm.VecSplat(0.8),
			// },
		},
		OrthographicProjection{
			Scale:          1,
			ViewportOrigin: gm.VecSplat(0.5),
			ScalingMode: ScalingModeAutoMin{
				MinWidth:  1000,
				MinHeight: 1000,
			},
		},
	)

	commands.Spawn(
		UiCamera,
		PickingCamera{},
		Camera{Order: 1},
		OrthographicProjection{
			Scale: 1,
			ScalingMode: ScalingModeFixedHorizontal{
				ViewportWidth: 800.0,
			},
			ViewportOrigin: gm.VecSplat(0.5),
		},
	)
}

var UiCamera = RenderLayersOf(1)
