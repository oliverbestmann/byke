package main

import (
	"math/rand/v2"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// A wall of colored tiles. Click or drag over a tile to knock it out, it falls and spins away.
// Port of the tiles example from github.com/ThreeDotsLabs/go-ecs-games (ebiten + donburi).

const (
	ScreenWidth  = 1024
	ScreenHeight = 960

	TileSize = 64
)

type Velocity struct {
	Component[Velocity]
	glm.Vec2f

	// rotation speed in radians per second
	Rotation glm.Rad
}

type Clickable struct {
	ImmutableComponent[Clickable]
}

var tileColors = []Color{
	ColorSRGB(0x00/255.0, 0x81/255.0, 0xa7/255.0),
	ColorSRGB(0x00/255.0, 0xaf/255.0, 0xb9/255.0),
	ColorSRGB(0xfd/255.0, 0xfc/255.0, 0xdc/255.0),
	ColorSRGB(0xfe/255.0, 0xd9/255.0, 0xb7/255.0),
	ColorSRGB(0xf0/255.0, 0x71/255.0, 0x67/255.0),
}

type Rng struct {
	*rand.Rand
}

func main() {
	var app App

	// fixed seed, so every run looks the same
	app.InsertResource(Rng{rand.New(rand.NewPCG(1, 2))})
	app.InsertResource(WindowConfig{Title: "Tiles", Width: ScreenWidth, Height: ScreenHeight})

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Click or drag over the tiles"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, velocitySystem)
	app.AddSystems(Update, System(clickSystem).RunIf(MouseButtonIsPressed(vyn.MouseButtonLeft)))

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, rng *Rng) {
	// origin at the top left of the window, y up, so the screen is at negative y
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{ViewportOrigin: glm.V2f(0, 1)},
	)

	// one white mesh for all tiles, tinted by the material
	tile := Rectangle(glm.V2f(TileSize, TileSize))

	for x := range ScreenWidth / TileSize {
		for y := range ScreenHeight / TileSize {
			commands.Spawn(
				Clickable{},
				Velocity{},
				TransformFromXY(float32(x*TileSize+TileSize/2), -float32(y*TileSize+TileSize/2)),
				MeshInstance{Mesh: tile},
				ColorMaterial{Color: tileColors[rng.IntN(len(tileColors))]},
			)
		}
	}
}

func velocitySystem(vt VirtualTime, query Query[struct {
	Velocity  Velocity
	Transform *Transform
}],
) {
	for item := range query.Items() {
		step := item.Velocity.Scale(vt.DeltaSecs)
		item.Transform.Translation = item.Transform.Translation.Add(step.Extend(0))
		*item.Transform = item.Transform.RotateZ(item.Velocity.Rotation * glm.Rad(vt.DeltaSecs))
	}
}

func clickSystem(
	commands *Commands,
	rng *Rng,
	cursor MouseCursor,
	tiles Query[struct {
		_ With[Clickable]
		EntityId
		Transform *Transform
		Velocity  *Velocity
	}],
) {
	// the cursor is in window pixels with y down, the camera has y up
	click := glm.V2f(cursor.Vec2f.X, -cursor.Vec2f.Y)

	for tile := range tiles.Items() {
		delta := tile.Transform.Translation.Truncate().Sub(click)
		if max(abs(delta.X), abs(delta.Y)) > TileSize/2 {
			continue
		}

		// the original moves per tick at 60 ticks per second, these are per second
		tile.Velocity.Vec2f = glm.V2f(rng.Range(-60, 60), -rng.Range(90, 150))
		tile.Velocity.Rotation = glm.Rad(rng.Range(-6, 6))

		// falling tiles fly in front of the wall
		tile.Transform.Translation.Z = 1

		commands.Entity(tile.EntityId).Remove[Clickable]()
	}
}

func abs(x float32) float32 {
	return max(x, -x)
}

func (rng *Rng) Range(min, max float32) float32 {
	return min + rng.Float32()*(max-min)
}
