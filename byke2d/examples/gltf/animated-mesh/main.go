package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Adapted from https://bevy.org/examples/animation/animated-mesh/
// Plays the run animation of a skinned glTF model.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, playRunAnimationSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 40})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(100, 100, 150).LookingAt(glm.Vec3f{0, 20, 0}, glm.Vec3f{0, 1, 0}),
	)

	commands.Spawn(
		NewTransform(),
		MeshInstance{Mesh: PlaneWithSize(glm.Vec2f{500, 500})},
		StandardMaterial{BaseColor: ColorSRGB(0.3, 0.5, 0.3), PerceptualRoughness: 1},
	)

	commands.Spawn(
		// the shadow volume starts at the light, so place it above the scene
		TransformFromXYZ(100, 200, 50).LookingAt(glm.Vec3f{}, glm.Vec3f{0, 1, 0}),
		DirectionalLight{
			Color:            ColorWhite,
			Illuminance:      DefaultDirectionalLight.Illuminance,
			ShadowsEnabled:   true,
			ShadowVolume:     glm.Vec3f{400, 400, 400},
			ShadowDepthBias:  DefaultDirectionalLight.ShadowDepthBias,
			ShadowNormalBias: DefaultDirectionalLight.ShadowNormalBias,
		},
	)

	// spawn the animations paused, playRunAnimationSystem starts the one we want
	commands.Spawn(SceneRoot{
		Handle:          assets.GLTF("Fox.glb").Await(),
		PauseAnimations: true,
	})
}

// Each animation of the scene is an entity with a Name and an ActiveAnimation.
func playRunAnimationSystem(query Query[struct {
	_         Added[ActiveAnimation]
	Name      Name
	Animation *ActiveAnimation
}],
) {
	for item := range query.Items() {
		if item.Name.Name == "Run" {
			item.Animation.Paused = false
			item.Animation.Repeat = true
		}
	}
}
