package main

import (
	"math"
	"math/rand/v2"
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// Asteroids. The ship and the rocks are drawn as outlines built with Path, and every
// rock gets its own jagged shape. A hit splits a rock in two smaller ones, until they
// are small enough to break apart. Hitting a rock with the ship starts the game over.

const (
	areaWidth  = 800
	areaHeight = 600

	shipRadius   = 18
	rockRadius   = 48
	minRockSize  = 12
	bulletSpeed  = 500
	thrust       = 300
	turnSpeed    = 4
	lineWidth    = 3
	startingRock = 3
)

type Velocity struct {
	Component[Velocity]
	glm.Vec2f
}

type Spin struct {
	ImmutableComponent[Spin]
	Speed glm.Rad
}

type Ship struct {
	Component[Ship]
	Angle glm.Rad
}

type Rock struct {
	ImmutableComponent[Rock]
	Radius float32
}

type Bullet struct {
	ImmutableComponent[Bullet]
}

// Wave counts the waves of rocks, every wave has one more rock.
type Wave struct {
	Number int
}

// fixed seed, so every run looks the same
var rng = rand.New(rand.NewPCG(1, 2))

func main() {
	var app App

	app.InsertResource(Wave{})

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Arrow keys to fly, space to shoot"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, steerSystem, moveSystem, spinSystem, shrinkSystem)
	app.AddSystems(Update, System(shootSystem).RunIf(KeyIsJustPressed(vyn.KeySpace)))
	app.AddSystems(Update, System(bulletHitSystem, shipHitSystem, nextWaveSystem).Chain())

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 120})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		HDR{},
		// smooths the thin outlines
		MSAA{},
		ClearColor{Color: ColorBlack},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeAutoMin{MinWidth: areaWidth, MinHeight: areaHeight},
		},
	)

	// an arrow pointing to the right, with a notch at the back
	var path Path
	path.MoveTo(glm.Vec2f{shipRadius, 0})
	path.LineTo(glm.Vec2f{-12, 12})
	path.LineTo(glm.Vec2f{-6, 0})
	path.LineTo(glm.Vec2f{-12, -12})
	path.Close()

	commands.Spawn(
		Ship{Angle: math.Pi / 2},
		Velocity{},
		NewTransform().WithRotationZ(math.Pi/2),
		MeshInstance{Mesh: outline(path, shipRadius)},
		ColorMaterial{Color: ColorLinearRGB(1, 4, 5)},
	)
}

// outline turns a closed path into a thin ring. The inner edge is the path scaled
// towards the origin, this works for shapes where the origin sees every edge.
func outline(path Path, radius float32) *Mesh {
	contour := path.ToContour(1)

	inner := make([]glm.Vec2f, len(contour))
	for idx, point := range contour {
		inner[idx] = point.Scale(1 - lineWidth/radius)
	}

	return Polygon(contour, inner)
}

func spawnRock(commands *Commands, pos glm.Vec2f, radius float32) {
	// a circle with a random radius at each corner
	var path Path
	for idx := range 12 {
		angle := float64(idx) / 12 * 2 * math.Pi
		dist := radius * (0.6 + 0.4*rng.Float32())
		point := glm.Vec2f{float32(math.Cos(angle)), float32(math.Sin(angle))}.Scale(dist)

		if idx == 0 {
			path.MoveTo(point)
		} else {
			path.LineTo(point)
		}
	}

	path.Close()

	// small rocks fly faster
	angle := rng.Float64() * 2 * math.Pi
	speed := (0.5 + rng.Float32()) * 2000 / radius

	commands.Spawn(
		Rock{Radius: radius},
		Velocity{Vec2f: glm.Vec2f{float32(math.Cos(angle)), float32(math.Sin(angle))}.Scale(speed)},
		Spin{Speed: glm.Rad(rng.Float32()*2 - 1)},
		TransformFromXY(pos[0], pos[1]),
		MeshInstance{Mesh: outline(path, radius)},
		ColorMaterial{Color: ColorLinearRGB(2, 2, 2)},
	)
}

func forward(angle glm.Rad) glm.Vec2f {
	return glm.Vec2f{float32(math.Cos(float64(angle))), float32(math.Sin(float64(angle)))}
}

func steerSystem(vt VirtualTime, keys Keys, ship Single[struct {
	Ship      *Ship
	Velocity  *Velocity
	Transform *Transform
}],
) {
	s := ship.Value

	if keys.IsPressed(vyn.KeyArrowLeft) {
		s.Ship.Angle += turnSpeed * glm.Rad(vt.DeltaSecs)
	}

	if keys.IsPressed(vyn.KeyArrowRight) {
		s.Ship.Angle -= turnSpeed * glm.Rad(vt.DeltaSecs)
	}

	s.Transform.Rotation = glm.RotationZQuat(s.Ship.Angle)

	if keys.IsPressed(vyn.KeyArrowUp) {
		s.Velocity.Vec2f = s.Velocity.Add(forward(s.Ship.Angle).Scale(thrust * vt.DeltaSecs))
	}

	// drag slows the ship down when it does not thrust
	s.Velocity.Vec2f = s.Velocity.Scale(1 - 0.5*vt.DeltaSecs)
}

func shootSystem(commands *Commands, ship Single[struct {
	Ship      Ship
	Velocity  Velocity
	Transform Transform
}],
) {
	dir := forward(ship.Value.Ship.Angle)
	pos := ship.Value.Transform.Translation.Truncate().Add(dir.Scale(shipRadius))

	commands.Spawn(
		Bullet{},
		Velocity{Vec2f: ship.Value.Velocity.Add(dir.Scale(bulletSpeed))},
		TransformFromXY(pos[0], pos[1]),
		MeshInstance{Mesh: Circle(3, 8)},
		ColorMaterial{Color: ColorLinearRGB(6, 4, 1)},
		DespawnAfter(time.Second),
	)
}

// moveSystem moves everything with a velocity. Leaving the area on one side
// enters it on the other.
func moveSystem(vt VirtualTime, query Query[struct {
	Velocity  Velocity
	Transform *Transform
}],
) {
	wrap := func(value, half float32) float32 {
		half += rockRadius
		return value - 2*half*float32(math.Floor(float64((value+half)/(2*half))))
	}

	for item := range query.Items() {
		pos := item.Transform.Translation.Add(item.Velocity.Scale(vt.DeltaSecs).Extend(0))
		item.Transform.Translation = glm.Vec3f{wrap(pos[0], areaWidth/2), wrap(pos[1], areaHeight/2), pos[2]}
	}
}

func spinSystem(vt VirtualTime, query Query[struct {
	Spin      Spin
	Transform *Transform
}],
) {
	for item := range query.Items() {
		*item.Transform = item.Transform.RotateZ(item.Spin.Speed * glm.Rad(vt.DeltaSecs))
	}
}

func bulletHitSystem(
	commands *Commands,
	bullets Query[struct {
		_ With[Bullet]
		EntityId
		Transform Transform
	}],
	rocks Query[struct {
		EntityId
		Rock      Rock
		Transform Transform
	}],
) {
	// a rock can only break once, even if two bullets hit it in the same frame
	broken := map[EntityId]bool{}

	for bullet := range bullets.Items() {
		for rock := range rocks.Items() {
			pos := rock.Transform.Translation.Truncate()
			if broken[rock.EntityId] || pos.Sub(bullet.Transform.Translation.Truncate()).Length() > rock.Rock.Radius {
				continue
			}

			broken[rock.EntityId] = true
			commands.Entity(bullet.EntityId).Despawn()
			commands.Entity(rock.EntityId).Despawn()

			if half := rock.Rock.Radius / 2; half >= minRockSize {
				spawnRock(commands, pos, half)
				spawnRock(commands, pos, half)
			} else {
				spawnDebris(commands, pos)
			}

			break
		}
	}
}

// spawnDebris spawns small pieces that fly apart and shrink.
func spawnDebris(commands *Commands, pos glm.Vec2f) {
	for idx := range 8 {
		dir := forward(glm.Rad(float32(idx) / 8 * 2 * math.Pi))

		commands.Spawn(
			Velocity{Vec2f: dir.Scale(120)},
			TransformFromXY(pos[0], pos[1]),
			MeshInstance{Mesh: Circle(2, 6)},
			ColorMaterial{Color: ColorLinearRGB(2, 2, 2)},
			DespawnAfter(500*time.Millisecond),
		)
	}
}

// The despawn timer also drives the size of each piece of debris.
func shrinkSystem(query Query[struct {
	_         Without[Bullet]
	Despawn   DespawnWithDelay
	Transform *Transform
}],
) {
	for item := range query.Items() {
		s := float32(item.Despawn.Timer.FractionRemaining())
		item.Transform.Scale = glm.Vec3f{s, s, 1}
	}
}

func shipHitSystem(
	commands *Commands,
	wave *Wave,
	ship Single[struct {
		Ship      *Ship
		Velocity  *Velocity
		Transform *Transform
	}],
	rocks Query[struct {
		EntityId
		Rock      Rock
		Transform Transform
	}],
) {
	shipPos := ship.Value.Transform.Translation.Truncate()

	crashed := false
	for rock := range rocks.Items() {
		dist := rock.Transform.Translation.Truncate().Sub(shipPos).Length()
		crashed = crashed || dist < rock.Rock.Radius+shipRadius/2
	}

	if !crashed {
		return
	}

	// start over: clear the rocks and put the ship back into the center
	for rock := range rocks.Items() {
		commands.Entity(rock.EntityId).Despawn()
	}

	spawnDebris(commands, shipPos)

	*wave = Wave{}
	*ship.Value.Ship = Ship{Angle: math.Pi / 2}
	*ship.Value.Velocity = Velocity{}
	*ship.Value.Transform = NewTransform().WithRotationZ(math.Pi / 2)
}

func nextWaveSystem(commands *Commands, wave *Wave, rocks Query[struct{ _ With[Rock] }]) {
	if rocks.Count() > 0 {
		return
	}

	wave.Number++

	// the rocks start on a circle around the ship
	for range startingRock + wave.Number - 1 {
		pos := forward(glm.Rad(rng.Float32() * 2 * math.Pi)).Scale(260)
		spawnRock(commands, pos, rockRadius)
	}
}
