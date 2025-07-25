package main

import (
	"embed"
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
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
	commands.Spawn(Camera{})

	// commands.Spawn(Camera{}, NewTransform().
	// 	WithRotation(math.Pi/2).
	// 	WithScale(gm.VecSplat(0.25)))
}
