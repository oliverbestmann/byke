package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

var _ = ValidateComponent[OrbitingLight]()

// OrbitingLight marks the light that circles around the box.
type OrbitingLight struct {
	Component[OrbitingLight]
}

var boxCenter = glm.Vec3f{0, 0.5, 0}

var pointLight = PointLight{
	Color:     ColorLinearRGB(1, 1, 1),
	Intensity: 10000,
	Range:     20,

	ShadowsEnabled:   true,
	ShadowDepthBias:  DefaultPointLight.ShadowDepthBias,
	ShadowNormalBias: DefaultPointLight.ShadowNormalBias,
}

var spotLight = SpotLight{
	Color:      ColorLinearRGB(1, 1, 1),
	Intensity:  10000,
	Range:      20,
	InnerAngle: glm.DegToRad(15),
	OuterAngle: glm.DegToRad(25),

	ShadowsEnabled:   true,
	ShadowDepthBias:  DefaultSpotLight.ShadowDepthBias,
	ShadowNormalBias: DefaultSpotLight.ShadowNormalBias,
}

var directionalLight = DirectionalLight{
	Color:       ColorLinearRGB(1, 1, 1),
	Illuminance: 400,

	ShadowsEnabled:   true,
	ShadowVolume:     glm.Vec3f{10, 10, 1000},
	ShadowDepthBias:  DefaultDirectionalLight.ShadowDepthBias,
	ShadowNormalBias: DefaultDirectionalLight.ShadowNormalBias,
}

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint(shared.HintFirstPerson + ", space to switch the light"))
	app.AddPlugin(shared.PluginRotatable)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, orbitLightSystem, toggleLightSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera3d,
		Camera{},
		HDR{},
		FirstPersonViewController{Pitch: -0.35, Yaw: -0.25},
		PerspectiveProjection{Fov: glm.DegToRad(50), Near: 0.1},
		TransformFromXYZ(-2.5, 4.5, 9.0),
	)

	commands.Spawn(
		TransformFromXYZ(boxCenter.XYZ()),
		MeshInstance{Mesh: Cube()},
		StandardMaterial{BaseColor: ColorSRGB(0.5, 0.6, 1.0), PerceptualRoughness: 0.8},
	)

	commands.Spawn(
		NewTransform().WithRotationX(glm.DegToRad(-90)),
		MeshInstance{Mesh: Circle(4.0, 64)},
		StandardMaterial{
			BaseColor:           ColorSRGB(1, 1, 1),
			DoubleSided:         true,
			PerceptualRoughness: 0.5,
			Metallic:            0.5,
		},
	)

	commands.Spawn(
		TransformFromXYZ(4, 4, 0),
		OrbitingLight{},
		spotLight,

		// a glowing sphere to show where the light is.
		// It must not cast shadows, it would block the light it surrounds.
		MeshInstance{Mesh: Sphere(0.15, 32, 16)},
		StandardMaterial{
			BaseColor:     ColorSRGB(1, 1, 1),
			EmissiveScale: glm.Vec3f{4, 4, 4},
		},
		NotShadowCaster{},
	)
}

func orbitLightSystem(
	vt VirtualTime,
	query Query[struct {
		_         With[OrbitingLight]
		Transform *Transform
	}],
) {
	for item := range query.Items() {
		pos := glm.RotationYQuat(glm.Rad(vt.Elapsed.Seconds())).Transform(glm.Vec3f{4, 4, 0})
		item.Transform.Translation = pos

		// lights shine along local -z, point it at the box (does not matter for the point light)
		item.Transform.Rotation = glm.QuatFromRotationArc(glm.Vec3f{0, 0, -1}, boxCenter.Sub(pos).Normalize())
	}
}

func toggleLightSystem(
	commands *Commands,
	keys Keys,
	query Query[struct {
		_        With[OrbitingLight]
		EntityId EntityId
		IsPoint  Has[PointLight]
		IsSpot   Has[SpotLight]
	}],
) {
	if !keys.IsJustPressed(vyn.KeySpace) {
		return
	}

	// cycle point -> spot -> directional -> point
	for item := range query.Items() {
		entity := commands.Entity(item.EntityId)

		switch {
		case item.IsPoint.Exists():
			entity.Remove[PointLight]()
			entity.Insert(spotLight)

		case item.IsSpot.Exists():
			entity.Remove[SpotLight]()
			entity.Insert(directionalLight)

		default:
			entity.Remove[DirectionalLight]()
			entity.Insert(pointLight)
		}
	}
}
