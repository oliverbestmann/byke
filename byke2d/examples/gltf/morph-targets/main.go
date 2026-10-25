package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/animation/morph-targets/
// Plays a glTF animation that changes the weights of morph targets.

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.InsertResource(GlobalAmbientLight{Color: ColorWhite, Brightness: 150})

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, playWaveAnimationSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 40})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(3, 2.1, 10.2).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
	)

	commands.Spawn(
		NewTransform().WithRotationZ(glm.DegToRad(90)),
		DefaultDirectionalLight,
	)

	// spawn the animations paused, playWaveAnimationSystem starts the one we want
	commands.Spawn(SceneRoot{
		Handle:          assets.GLTF("MorphStressTest.glb").Await(),
		PauseAnimations: true,
	})
}

// Each animation of the scene is an entity with a Name and an ActiveAnimation.
func playWaveAnimationSystem(query Query[struct {
	_         Added[ActiveAnimation]
	Name      Name
	Animation *ActiveAnimation
}],
) {
	for item := range query.Items() {
		if item.Name.Name == "TheWave" {
			item.Animation.Paused = false
			item.Animation.Repeat = true
		}
	}
}
