package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
)

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.AddSystems(Update, addLightmapsToMeshesSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	box := assets.GLTF("CornellBox.glb").Await()

	commands.InsertResource(GlobalAmbientLightNone)

	commands.Spawn(
		Camera3d,
		Camera{},
		FirstPersonViewController{Velocity: 400},

		TransformFromXYZ(-278.0, 273.0, 800.0),
		DefaultPerspectiveProjection,
	)

	commands.Spawn(
		SceneRoot{Handle: box},
		NewTransform(),
	)
}

func addLightmapsToMeshesSystem(
	commands *Commands,
	assets *Assets,
	meshes Query[struct {
		_        With[MeshInstance]
		_        Without[Lightmap]
		EntityId EntityId
		Name     Name
		Material *StandardMaterial
	}],
) {
	// var exposure float32 = 250.0

	for item := range meshes.Items() {
		switch item.Name.Name {
		case "large_box":
			commands.Entity(item.EntityId).Insert(
				Lightmap{
					Image:   assets.Texture("CornellBox-Large.zstd.ktx2").Await(),
					Bicubic: false,
				},
			)
		case "small_box":
			commands.Entity(item.EntityId).Insert(
				Lightmap{
					Image:   assets.Texture("CornellBox-Small.zstd.ktx2").Await(),
					Bicubic: false,
				},
			)

		case "cornell_box":
			commands.Entity(item.EntityId).Insert(
				Lightmap{
					Image:   assets.Texture("CornellBox-Box.zstd.ktx2").Await(),
					Bicubic: false,
				},
			)
		}
	}
}
