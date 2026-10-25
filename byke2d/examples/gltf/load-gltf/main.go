package main

import (
	"math"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Adapted from https://bevy.org/examples/3d-rendering/load-gltf/
// Loads and renders a glTF file as a scene, lit by an environment map and a rotating sun.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, animateLightDirectionSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 100})
}

var helmetCenter = glm.Vec3f{0, 0.3, 0}

func setupSystem(commands *Commands, assets *Assets) {
	opts := &LoadTextureSettings{OverrideTextureViewDimension: wgpu.TextureViewDimensionCube}
	diffuse := assets.TextureWithSettings("pisa_diffuse_rgb9e5_zstd.ktx2", opts)
	specular := assets.TextureWithSettings("pisa_specular_rgb9e5_zstd.ktx2", opts)

	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		DefaultPerspectiveProjection,
		TransformFromXYZ(0.7, 0.7, 1.0).LookingAt(helmetCenter, glm.Vec3f{0, 1, 0}),
		EnvironmentMapLight{
			DiffuseMap:  diffuse.Await(),
			SpecularMap: specular.Await(),
			Intensity:   250,
		},
	)

	commands.Spawn(
		NewTransform(),
		DirectionalLight{
			Color:            ColorWhite,
			Illuminance:      DefaultDirectionalLight.Illuminance,
			ShadowsEnabled:   true,
			ShadowVolume:     glm.Vec3f{1.6, 1.6, 4},
			ShadowDepthBias:  DefaultDirectionalLight.ShadowDepthBias,
			ShadowNormalBias: DefaultDirectionalLight.ShadowNormalBias,
		},
	)

	commands.Spawn(SceneRoot{Handle: assets.GLTF("FlightHelmet/FlightHelmet.gltf").Await()})
}

func animateLightDirectionSystem(
	vt VirtualTime,
	light Single[struct {
		_         With[DirectionalLight]
		Transform *Transform
	}],
) {
	rot := glm.RotationYQuat(glm.Rad(vt.Elapsed.Seconds() * math.Pi / 5)).
		Mul(glm.RotationXQuat(glm.Rad(-math.Pi / 4)))

	// the shadow volume starts at the light, so move it back along its direction
	direction := rot.Transform(glm.Vec3f{0, 0, -1})

	light.Value.Transform.Rotation = rot
	light.Value.Transform.Translation = helmetCenter.Sub(direction.Scale(2))
}
