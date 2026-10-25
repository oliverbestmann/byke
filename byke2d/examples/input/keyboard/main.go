package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// Moves a square with the keyboard. IsPressed is true while a key is held,
// IsJustPressed only in the frame the key goes down. A run condition can also
// check a key, so the system only runs when it is pressed.

type Player struct {
	ImmutableComponent[Player]
}

var colors = []Color{
	ColorSRGB(1, 0.4, 0.4),
	ColorSRGB(0.4, 1, 0.4),
	ColorSRGB(0.4, 0.6, 1),
}

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("WASD or arrow keys to move, shift to run, space to change the color"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, moveSystem)
	app.AddSystems(Update, System(changeColorSystem).RunIf(KeyIsJustPressed(vyn.KeySpace)))

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{ViewportOrigin: glm.Vec2f{0.5, 0.5}},
	)

	commands.Spawn(
		Player{},
		NewTransform(),
		MeshInstance{Mesh: Rectangle(glm.Vec2f{64, 64})},
		ColorMaterial{Color: colors[0]},
	)
}

func moveSystem(
	vt VirtualTime,
	keys Keys,
	player Single[struct {
		_         With[Player]
		Transform *Transform
	}],
) {
	var dir glm.Vec2f

	if keys.IsPressed(vyn.KeyA) || keys.IsPressed(vyn.KeyArrowLeft) {
		dir[0] -= 1
	}

	if keys.IsPressed(vyn.KeyD) || keys.IsPressed(vyn.KeyArrowRight) {
		dir[0] += 1
	}

	if keys.IsPressed(vyn.KeyW) || keys.IsPressed(vyn.KeyArrowUp) {
		dir[1] += 1
	}

	if keys.IsPressed(vyn.KeyS) || keys.IsPressed(vyn.KeyArrowDown) {
		dir[1] -= 1
	}

	if dir == (glm.Vec2f{}) {
		return
	}

	speed := float32(200)
	if keys.IsPressed(vyn.KeyShiftLeft) || keys.IsPressed(vyn.KeyShiftRight) {
		speed = 500
	}

	step := dir.Normalize().Scale(speed * vt.DeltaSecs)
	player.Value.Transform.Translation = player.Value.Transform.Translation.Add(step.Extend(0))
}

func changeColorSystem(
	player Single[struct {
		_        With[Player]
		Material *ColorMaterial
	}],
	index *Local[int],
) {
	index.Value = (index.Value + 1) % len(colors)
	player.Value.Material.Color = colors[index.Value]
}
