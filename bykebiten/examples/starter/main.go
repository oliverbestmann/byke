package main

import (
	"fmt"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
)

func main() {
	var app App

	// Add the bykebiten game plugin
	app.AddPlugin(GamePlugin)

	// Add a WindowConfig to set title and initial size
	app.InsertResource(WindowConfig{
		Title:  "AssetDucky",
		Width:  800,
		Height: 600,
	})

	// configure plugins
	app.AddPlugin(PluginFunc(pluginScreen))
	app.AddPlugin(PluginFunc(pluginPause))
	app.AddPlugin(PluginFunc(pluginMenu))
	app.AddPlugin(PluginFunc(pluginGame))

	fmt.Println(app.Run())
}
