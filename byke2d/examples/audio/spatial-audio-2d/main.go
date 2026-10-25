package main

import (
	"math"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// Adapted from https://bevy.org/examples/audio/spatial-audio-2d/
// A sound circles the listener. The volume of each ear follows its distance to the sound.

const (
	// half the distance between the ears, in pixels
	earGap = 40

	// one world unit is 100 pixels. The sound plays at full volume within one unit
	spatialScale = 0.01

	orbitRadius = 200
)

type Emitter struct {
	ImmutableComponent[Emitter]
}

type Listener struct {
	ImmutableComponent[Listener]
}

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Arrow keys to move the listener"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, orbitEmitterSystem, moveListenerSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 60})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{ViewportOrigin: glm.Vec2f{0.5, 0.5}},
	)

	// the sound source, a spatial looping sound
	commands.Spawn(
		Emitter{},
		TransformFromXY(orbitRadius, 0),
		AudioPlayerOf(assets.Audio("engine.ogg").Await()),
		PlaybackSettingsLoop.WithSpatialScale(glm.Vec3f{1, 1, 1}.Scale(spatialScale)),
		MeshInstance{Mesh: Circle(16, 32)},
		ColorMaterial{Color: ColorSRGB(0.3, 0.6, 1)},
	)

	// the listener, with one ear to the left and one to the right
	listener := commands.Spawn(
		Listener{},
		NewTransform(),
		Microphone{
			LeftEarOffset:  glm.Vec3f{-earGap, 0, 0},
			RightEarOffset: glm.Vec3f{earGap, 0, 0},
		},
		MeshInstance{Mesh: Circle(24, 32)},
		ColorMaterial{Color: ColorWhite},
	).Id()

	for _, x := range []float32{-earGap, earGap} {
		commands.Spawn(
			ChildOf{Parent: listener},
			TransformFromXYZ(x, 0, 0.1),
			MeshInstance{Mesh: Circle(8, 16)},
			ColorMaterial{Color: ColorSRGB(1, 0.3, 0.3)},
		)
	}
}

func orbitEmitterSystem(
	vt VirtualTime,
	emitter Single[struct {
		_         With[Emitter]
		Transform *Transform
	}],
) {
	angle := vt.Elapsed.Seconds()
	emitter.Value.Transform.Translation = glm.Vec3f{
		float32(math.Cos(angle)) * orbitRadius,
		float32(math.Sin(angle)) * orbitRadius,
		0,
	}
}

func moveListenerSystem(
	vt VirtualTime,
	keys Keys,
	listener Single[struct {
		_         With[Listener]
		Transform *Transform
	}],
) {
	var dir glm.Vec2f

	if keys.IsPressed(vyn.KeyArrowLeft) {
		dir[0] -= 1
	}

	if keys.IsPressed(vyn.KeyArrowRight) {
		dir[0] += 1
	}

	if keys.IsPressed(vyn.KeyArrowUp) {
		dir[1] += 1
	}

	if keys.IsPressed(vyn.KeyArrowDown) {
		dir[1] -= 1
	}

	step := dir.Scale(300 * float32(vt.DeltaSecs))
	listener.Value.Transform.Translation[0] += step[0]
	listener.Value.Transform.Translation[1] += step[1]
}
