package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	. "github.com/oliverbestmann/byke/gm"
	"math"
	"math/rand/v2"
	"time"
)

func main() {
	var app App

	app.InsertResource(WindowConfig{
		Title:  "Asteroids",
		Width:  800,
		Height: 600,
	})

	var InputSystems = &SystemSet{}
	var PhysicsSystems = &SystemSet{}

	app.ConfigureSystemSets(Update, InputSystems.Before(PhysicsSystems))

	app.AddPlugin(GamePlugin)

	app.InsertResource(Gravity(Vec{Y: -9.81}))

	app.AddSystems(Startup, setupCamera, spawnSpaceShipSystem, spawnLevelSystem)
	app.AddSystems(Update, System(handleSpaceshipInput).InSet(InputSystems))
	app.AddSystems(Update, System(applyGravitySystem, moveObjectsSystem, checkGroundCollisionSystem).Chain().InSet(PhysicsSystems))
	app.AddSystems(PostUpdate, followShipSystem, alignWithVelocity, despawnWithDelaySystem)
	fmt.Println(app.Run())
}

type Gravity Vec

var _ = ValidateComponent[SpaceShip]()
var _ = ValidateComponent[Plume]()
var _ = ValidateComponent[Missile]()
var _ = ValidateComponent[DespawnWithDelay]()
var _ = ValidateComponent[AlignWithVelocity]()

type SpaceShip struct {
	Component[SpaceShip]
}

type Plume struct {
	Component[Plume]
}

type Velocity struct {
	Component[Velocity]
	Linear Vec
}

type Missile struct {
	Component[Missile]
}

type AlignWithVelocity struct {
	Component[AlignWithVelocity]
}

type DespawnWithDelay struct {
	Component[DespawnWithDelay]
	Timer Timer
}

func setupCamera(commands *Commands) {
	commands.Spawn(
		TransformFromXY(0, 300).WithScale(Vec{X: -1, Y: -1}),
		Camera{},
		OrthographicProjection{
			ViewportOrigin: VecSplat(0.5),
			ScalingMode:    ScalingModeFixedVertical{ViewportHeight: 600},
			Scale:          1,
		},
	)
}

var shipCorners = []Vec{
	{X: -10, Y: 10},
	{X: 15, Y: 0},
	{X: -10, Y: -10},
}

func spawnSpaceShipSystem(commands *Commands) {
	var spaceShipShape Path
	for _, vec := range shipCorners {
		spaceShipShape.LineTo(vec)
	}
	spaceShipShape.Close()

	var plume Path
	plume.MoveTo(Vec{X: -10, Y: 5})
	plume.LineTo(Vec{X: -10, Y: -5})
	plume.LineTo(Vec{X: -20, Y: 0})
	plume.Close()

	commands.Spawn(
		SpaceShip{},
		TransformFromXY(0, 300),
		Velocity{},
		spaceShipShape,
		Stroke{
			Width: 2,
			Color: color.White,
		},

		SpawnChild(
			Plume{},
			plume,
			// put the plume below the spaceship
			Layer{Z: -0.1},
			Fill{
				Color: color.RGB(1, 0.75, 0.5),
			},
		),
	)
}

type Heightmap struct {
	height []Vec
}

func (h *Heightmap) IsAboveGround(p Vec) bool {
	prev := h.height[0]

	for _, next := range h.height[1:] {
		if p.X < prev.X || p.X > next.X {
			prev = next
			continue
		}

		return next.Sub(prev).Cross(p.Sub(prev)) > 0
	}

	return p.Y > 0
}

func spawnLevelSystem(commands *Commands) {
	var height []Vec

	var terrain Path
	for x := -2000.0; x <= 2000; x += 200 {
		pos := Vec{
			X: rand.Float64()*50 - 25 + x,
			Y: rand.Float64()*100 + 20,
		}

		terrain.LineTo(pos)
		height = append(height, pos)
	}

	// store the heightmap for later collision checking
	commands.InsertResource(Heightmap{height: height})

	commands.Spawn(
		terrain,
		Stroke{
			Width: 4,
			Color: color.Gray(0.7),
		},
	)
}

func checkGroundCollisionSystem(
	commands *Commands,

	heightmap Heightmap,

	ship Single[struct {
	_ With[SpaceShip]
	EntityId
	Transform Transform
}],
) {
	s := &ship.Value

	for _, vec := range shipCorners {
		vec = vec.Rotate(s.Transform.Rotation)
		vec = vec.Add(s.Transform.Translation)

		above := heightmap.IsAboveGround(vec)
		if !above {
			commands.Entity(s.EntityId).Despawn()
			commands.Queue(Explode(vec))
			break
		}
	}
}

func handleSpaceshipInput(commands *Commands, keys Keys, vt VirtualTime,
	ship Single[struct {
	_         With[SpaceShip]
	Transform *Transform
	Velocity  *Velocity
}],
	plume Single[struct {
	_          With[Plume]
	Fill       *Fill
	Visibility *Visibility
}],
) {
	s := &ship.Value
	p := &plume.Value

	if keys.IsPressed(ebiten.KeyLeft) {
		s.Transform.Rotation -= DegToRad(270) * Rad(vt.DeltaSecs)
	}

	if keys.IsPressed(ebiten.KeyRight) {
		s.Transform.Rotation += DegToRad(270) * Rad(vt.DeltaSecs)
	}

	if keys.IsPressed(ebiten.KeyUp) {
		delta := RotationMat(s.Transform.Rotation).Transform(Vec{X: 250})
		s.Velocity.Linear = s.Velocity.Linear.Add(delta.Mul(vt.DeltaSecs))
		p.Visibility.SetVisible()
	} else {
		p.Visibility.SetInvisible()

	}

	// Maybe limit maximum velocity?
	// if l := s.Velocity.Linear.Length(); l > 300 {
	// 	s.Velocity.Linear = s.Velocity.Linear.Mul(300 / l)
	// }

	if keys.IsJustPressed(ebiten.KeySpace) {
		velocity := RotationMat(s.Transform.Rotation).Transform(Vec{X: 500})

		commands.Queue(FireMissile(
			s.Transform.Translation.Add(velocity.Normalized().Mul(10)),
			s.Velocity.Linear.Add(velocity),
		))
	}
}

func applyGravitySystem(vt VirtualTime, gravity Gravity, query Query[*Velocity]) {
	gravityScaled := Vec(gravity).Mul(vt.DeltaSecs)
	for velocity := range query.Items() {
		velocity.Linear = velocity.Linear.Add(gravityScaled)
	}
}

func moveObjectsSystem(vt VirtualTime, query Query[struct {
	Velocity  Velocity
	Transform *Transform
}]) {
	for item := range query.Items() {
		item.Transform.Translation = item.Transform.Translation.Add(item.Velocity.Linear.Mul(vt.DeltaSecs))
	}
}

func followShipSystem(vt VirtualTime,
	camera Single[struct {
	_         With[Camera]
	Transform *Transform
}],
	ship Single[struct {
	_         With[SpaceShip]
	Transform Transform
	Velocity  Velocity
}],
) {
	pos := ship.Value.Transform.Translation

	targetX := pos.X + ship.Value.Velocity.Linear.X
	targetY := max(300, pos.Y+ship.Value.Velocity.Linear.Y)

	posCamera := &camera.Value.Transform.Translation

	x := nudge(targetX, posCamera.X, 2, vt.DeltaSecs)
	y := nudge(targetY, posCamera.Y, 2, vt.DeltaSecs)

	posCamera.X = moveTowards(posCamera.X, targetX, x)
	posCamera.Y = moveTowards(posCamera.Y, targetY, y)
}

func nudge(target, current, decay, dt float64) float64 {
	return (target - current) * (1 - math.Exp(-dt*decay))
}

func moveTowards(current, target, delta float64) float64 {
	result := target

	if current < target {
		result = min(current+math.Abs(delta), target)
	}

	if current > target {
		result = max(current-math.Abs(delta), target)
	}

	return result
}

func Explode(pos Vec) Command {
	return func(world *World) {
		world.RunSystem(func(commands *Commands) {
			var circle Path
			circle.Circle(VecZero, 1)

			commands.Spawn(
				DespawnAfter(100*time.Millisecond),
				circle,
				TransformFromXY(pos.XY()).WithScale(VecSplat(50.0)),
				Fill{Color: color.RGB(1.0, 0.5, 0.2)},
				Layer{Z: 1},
			)

			commands.Spawn(
				DespawnAfter(150*time.Millisecond),
				circle,
				TransformFromXY(pos.XY()).WithScale(VecSplat(50.0)),
				Stroke{
					Width: 5,
					Color: color.RGBA(1.0, 0.5, 0.2, 0.5),
				},
				Layer{Z: 2},
			)
		})
	}
}

func FireMissile(start, velocity Vec) Command {
	return func(world *World) {
		world.RunSystem(func(commands *Commands) {
			var missile Path
			missile.MoveTo(Vec{X: -5})
			missile.LineTo(Vec{X: 5})

			commands.Spawn(
				TransformFromXY(start.XY()).WithRotation(velocity.Angle()),
				Missile{},
				AlignWithVelocity{},
				Velocity{Linear: velocity},
				missile,
				Stroke{Width: 2, Color: color.White},
				DespawnAfter(10*time.Second),
			)
		})
	}
}

func alignWithVelocity(
	query Query[struct {
	_         With[AlignWithVelocity]
	Velocity  Velocity
	Transform *Transform
}],
) {
	for item := range query.Items() {
		item.Transform.Rotation = item.Velocity.Linear.Angle()
	}
}

func DespawnAfter(duration time.Duration) DespawnWithDelay {
	return DespawnWithDelay{
		Timer: NewTimer(duration, TimerModeOnce),
	}
}

func despawnWithDelaySystem(
	commands *Commands,
	vt VirtualTime,
	query Query[struct {
	EntityId
	DespawnWithDelay *DespawnWithDelay
}],
) {
	for item := range query.Items() {
		timer := &item.DespawnWithDelay.Timer
		if timer.Tick(vt.Delta).JustFinished() || timer.Finished() {
			commands.Entity(item.EntityId).Despawn()
		}
	}
}
