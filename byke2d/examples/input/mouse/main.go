package main

import (
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// A circle follows the mouse cursor. Clicking drops dots that shrink and disappear,
// holding the right button makes the circle bigger.

type Follower struct {
	ImmutableComponent[Follower]
}

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Click to drop dots, hold the right button to grow"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, followCursorSystem, shrinkDotsSystem)
	app.AddSystems(Update, System(dropDotSystem).RunIf(MouseButtonIsPressed(vyn.MouseButtonLeft)))

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{ViewportOrigin: glm.Vec2f{0.5, 0.5}},
	)

	commands.Spawn(
		Follower{},
		TransformFromXYZ(0, 0, 1),
		MeshInstance{Mesh: Circle(24, 32)},
		ColorMaterial{Color: ColorWhite},
	)
}

// cursorToWorld converts the cursor from window pixels (origin at the top left, y down)
// to the coordinates of the centered camera (origin in the middle, y up).
func cursorToWorld(cursor MouseCursor, screen ScreenSize) glm.Vec2f {
	return glm.Vec2f{cursor.Vec2f[0] - screen.Vec2f[0]/2, screen.Vec2f[1]/2 - cursor.Vec2f[1]}
}

func followCursorSystem(
	cursor MouseCursor,
	screen ScreenSize,
	buttons MouseButtons,
	follower Single[struct {
		_         With[Follower]
		Transform *Transform
	}],
) {
	tr := follower.Value.Transform
	tr.Translation = cursorToWorld(cursor, screen).Extend(1)

	tr.Scale = glm.Vec3f{1, 1, 1}
	if buttons.IsPressed(vyn.MouseButtonRight) {
		tr.Scale = glm.Vec3f{2, 2, 1}
	}
}

func dropDotSystem(commands *Commands, cursor MouseCursor, screen ScreenSize) {
	pos := cursorToWorld(cursor, screen)

	commands.Spawn(
		TransformFromXY(pos[0], pos[1]),
		MeshInstance{Mesh: Circle(12, 16)},
		ColorMaterial{Color: ColorSRGB(1, 0.5, 0.2)},
		DespawnAfter(time.Second),
	)
}

// The despawn timer also drives the size of each dot.
func shrinkDotsSystem(query Query[struct {
	Despawn   DespawnWithDelay
	Transform *Transform
}],
) {
	for item := range query.Items() {
		s := float32(item.Despawn.Timer.FractionRemaining())
		item.Transform.Scale = glm.Vec3f{s, s, 1}
	}
}
