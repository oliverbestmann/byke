package main

import (
	"embed"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	. "github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/partycle"
	"github.com/oliverbestmann/byke/physics"
	"github.com/pkg/profile"
)

//go:embed assets/*
var assets embed.FS

func main() {
	// defer profile.Start(profile.MemProfile, profile.MemProfileRate(512)).Stop()
	defer profile.Start(profile.CPUProfile).Stop()

	var app App

	app.InsertResource(WindowConfig{
		Title:  "Asteroids",
		Width:  800,
		Height: 600,
	})

	var InputSystems = &SystemSet{}
	var GameSystems = &SystemSet{}

	app.InsertResource(MakeAssetFS(assets))

	app.ConfigureSystemSets(Update, InputSystems.Before(GameSystems))

	app.AddPlugin(GamePlugin)
	app.AddPlugin(physics.Plugin)
	app.AddPlugin(partycle.Plugin)

	app.InsertResource(GlobalVolume{Volume: 0.1})
	app.InsertResource(GlobalSpatialScale{Scale: VecSplat(1 / 500.0)})

	app.AddSystems(Startup, setupCamera, spawnSpaceShipSystem, spawnTerrainSystem)
	app.AddSystems(PreUpdate, pauseSystem)
	app.AddSystems(FixedUpdate, applyGravityToShipSystem)
	app.AddSystems(Update, System(handleSpaceshipInput).InSet(InputSystems))
	app.AddSystems(Update, System(spawnSmokeSystem).InSet(GameSystems))
	app.AddSystems(PostUpdate, System(moveCameraTargetSystem, moveCameraSystem).Chain())

	app.World().AddObserver(NewObserver(spawnExplosionSystem))

	// preload audio assets in the background
	app.World().RunSystem(func(assets *Assets) {
		assets.Audio("launch.ogg")
		assets.Audio("explosion.ogg")
	})

	fmt.Println(app.Run())
}

var _ = ValidateComponent[SpaceShip]()
var _ = ValidateComponent[Plume]()
var _ = ValidateComponent[Missile]()
var _ = ValidateComponent[SmokeEmitter]()
var _ = ValidateComponent[CameraTarget]()

type SpaceShip struct {
	Component[SpaceShip]
}

type Plume struct {
	Component[Plume]
}

type Missile struct {
	Component[Missile]
}

type SmokeEmitter struct {
	Component[SmokeEmitter]
	Offset   Vec
	Velocity Vec
	Timer    Timer
}

type CameraTarget struct {
	ImmutableComponent[CameraTarget]
}

type TerrainContact struct {
	Position Vec
}

func setupCamera(commands *Commands) {
	commands.Spawn(
		TransformFromXY(0, 300).WithScale(-1, -1),
		Camera{},

		// listen left and right of the camera
		Microphone{
			LeftEarOffset:  Vec{X: 20},
			RightEarOffset: Vec{Y: -20},
		},

		OrthographicProjection{
			ViewportOrigin: VecSplat(0.5),
			ScalingMode:    ScalingModeFixedVertical{ViewportHeight: 600},
			Scale:          1,
		},
	)

	var path Path
	path.Circle(VecZero, 3)

	commands.Spawn(
		TransformFromXY(0, 300),
		CameraTarget{},
		// path,
		// Fill{Color: color.RGB(1, 0, 0)},
	)
}

func spawnSpaceShipSystem(commands *Commands) {
	var shipCorners = []Vec{
		{X: -10, Y: 10},
		{X: 15, Y: 0},
		{X: -10, Y: -10},
	}

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

	commands.
		Spawn(
			SpaceShip{},
			TransformFromXY(0, 300),
			physics.RigidBodyKinematic,
			physics.Collider{
				Shape: physics.PolygonShape{
					Points: shipCorners,
				},
			},

			spaceShipShape,
			physics.Sensor{},
			physics.CollisionEventsEnabled{},

			Fill{Color: color.Black},

			Stroke{
				Width:     2,
				Color:     color.White,
				Antialias: true,
			},

			SpawnChild(
				Plume{},
				plume,
				// put the plume below the spaceship
				Layer{Z: -0.1},
				Fill{
					Color:     color.RGB(1, 0.75, 0.5),
					Antialias: true,
				},
			),
		).
		Observe(func(trigger On[TerrainContact], commands *Commands) {
			commands.Entity(trigger.Target).Despawn()
			commands.Trigger(Explode{Position: trigger.Event.Position, Radius: 50})
		}).
		Observe(func(trigger On[physics.OnCollisionStarted], commands *Commands) {
			commands.Entity(trigger.Target).Despawn()
			commands.Trigger(Explode{Position: trigger.Event.Position, Radius: 50})
		})
}

type Terrain struct {
	height []Vec
}

func (h *Terrain) IsAboveGround(p Vec) bool {
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

func spawnTerrainSystem(commands *Commands) {
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

	// fill in the ground
	terrain.LineTo(Vec{X: 2000, Y: -10})
	terrain.LineTo(Vec{X: -2000, Y: -10})

	// store the heightmap for later collision checking
	commands.InsertResource(Terrain{height: height})

	commands.Spawn(
		terrain,
		Layer{Z: 10},
		Fill{Color: color.Black},
		Stroke{
			Width:     4,
			Color:     color.Gray(0.7),
			Antialias: true,
		},
	)

	// spawn one collider per segment
	for idx := 0; idx < len(height)-1; idx++ {
		points := []Vec{
			height[idx],
			height[idx+1],
			{X: height[idx+1].X, Y: 0},
			{X: height[idx].X, Y: 0},
		}

		commands.Spawn(
			physics.RigidBodyStatic,
			physics.Collider{
				Shape: physics.PolygonShape{
					Points: points,
				},
			},
		)
	}
}

func pauseSystem(vt *VirtualTime, keys Keys) {
	if keys.IsJustPressed(ebiten.KeyP) {
		if vt.Scale == 0 {
			vt.Scale = 1
		} else {
			vt.Scale = 0
		}
	}
}

func handleSpaceshipInput(commands *Commands, keys Keys, vt VirtualTime,
	ship Single[struct {
		_ With[SpaceShip]
		EntityId
		Transform *Transform
		Velocity  *physics.Velocity
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

		circle := Circle(0.1, 12)
		commands.Entity(s.EntityId).Update(InsertComponent(partycle.Emitter{
			ParticlesPerSecond:       70,
			ParticlesPerSecondJitter: 30,
			LinearVelocity:           Vec{X: -100},
			LinearVelocityJitter:     Vec{X: 20},
			AngularVelocityJitter:    math.Pi * 1,
			RotationJitter:           math.Pi * 2,
			DampeningLinear:          1,
			DampeningAngular:         1,
			ParticleLifetime:         200 * time.Millisecond,
			ParticleLifetimeJitter:   40 * time.Millisecond,
			Visual: func() ErasedComponent {
				return circle
			},
		}))
	} else {
		p.Visibility.SetInvisible()

		commands.Entity(s.EntityId).Update(RemoveComponent[partycle.Emitter]())
	}

	if keys.IsJustPressed(ebiten.KeySpace) {
		velocity := RotationMat(s.Transform.Rotation).Transform(Vec{X: 500})

		commands.Queue(FireMissile(
			s.Transform.Translation.Add(velocity.Normalized().Mul(25)),
			s.Velocity.Linear.Add(velocity),
		))
	}
}

func applyGravityToShipSystem(
	vt VirtualTime,
	gravity physics.Gravity,
	velocities Query[*physics.Velocity],
) {
	for vel := range velocities.Items() {
		vel.Linear = vel.Linear.Add(gravity.Value.Mul(vt.DeltaSecs))
	}
}

func moveCameraTargetSystem(
	vt VirtualTime,
	cameraTarget Single[struct {
		_         With[CameraTarget]
		Transform *Transform
	}],
	ship Single[struct {
		_         With[SpaceShip]
		Transform Transform
		Velocity  physics.Velocity
	}],
) {
	posShip := ship.Value.Transform.Translation

	target := posShip.Add(ship.Value.Velocity.Linear)
	target.Y = max(300, target.Y)

	posCameraTarget := &cameraTarget.Value.Transform.Translation

	x := nudge(target.X, posCameraTarget.X, 5, vt.DeltaSecs)
	y := nudge(target.Y, posCameraTarget.Y, 5, vt.DeltaSecs)

	posCameraTarget.X = moveTowards(posCameraTarget.X, target.X, x)
	posCameraTarget.Y = moveTowards(posCameraTarget.Y, target.Y, y)

	delta := posCameraTarget.Sub(posShip)
	if delta.Length() > 300 {
		*posCameraTarget = posShip.Add(delta.Normalized().Mul(300))
	}
}

func moveCameraSystem(
	vt VirtualTime,
	camera Single[struct {
		_         With[Camera]
		Transform *Transform
	}],
	target Single[struct {
		_         With[CameraTarget]
		Transform Transform
	}],
) {
	posTarget := target.Value.Transform.Translation

	posCamera := &camera.Value.Transform.Translation

	x := nudge(posTarget.X, posCamera.X, 5, vt.DeltaSecs)
	y := nudge(posTarget.Y, posCamera.Y, 5, vt.DeltaSecs)

	posCamera.X = moveTowards(posCamera.X, posTarget.X, x)
	posCamera.Y = moveTowards(posCamera.Y, posTarget.Y, y)
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

type Explode struct {
	Position Vec
	Radius   float64
}

func spawnExplosionSystem(params On[Explode], commands *Commands, assets *Assets) {
	p := &params.Event

	var circle Path
	circle.Circle(VecZero, p.Radius)

	// just for fun, we can use a mesh here
	commands.Spawn(
		DespawnAfter(100*time.Millisecond),
		Circle(p.Radius, 48),
		TransformFromXY(p.Position.XY()),
		ColorTint{Color: color.RGB(1.0, 0.5, 0.2)},
		Layer{Z: 1},
	)

	// // we can use the path we've prepared above
	// 	commands.Spawn(
	// 		DespawnAfter(100*time.Millisecond),
	// 		circle,
	// 		TransformFromXY(p.Position.XY()),
	// 		Fill{Color: color.RGB(1.0, 0.5, 0.2)},
	// 		Layer{Z: 1},
	// 	)

	commands.Spawn(
		DespawnAfter(150*time.Millisecond),
		circle,
		TransformFromXY(p.Position.XY()),
		Stroke{
			Width:     20,
			Color:     color.RGBA(1.0, 0.5, 0.2, 0.5),
			Antialias: true,
		},
		Layer{Z: 2},
	)

	commands.Spawn(
		AudioPlayerOf(assets.Audio("explosion.ogg").Await()),
		PlaybackSettingsDespawn.WithStartAt(900*time.Millisecond).WithSpatial(),
		TransformFromXY(p.Position.XY()),
	)
}

type FireMissileIn struct {
	Start    Vec
	Velocity Vec
}

func FireMissile(start, velocity Vec) Command {
	return CommandFn(func(world *World) {
		world.RunSystemWithInValue(fireMissileSystem, FireMissileIn{Start: start, Velocity: velocity})
	})
}

func fireMissileSystem(commands *Commands, assets *Assets, param In[FireMissileIn]) {
	p := &param.Value

	var missile Path
	missile.MoveTo(Vec{X: -5})
	missile.LineTo(Vec{X: 5})

	commands.
		Spawn(
			TransformFromXY(p.Start.XY()).WithRotation(p.Velocity.Angle()),
			Missile{},
			physics.Collider{
				Shape: physics.SegmentShape{
					A:      Vec{X: -5},
					B:      Vec{X: 5},
					Radius: 3,
				},
			},
			physics.RigidBodyDynamic,
			physics.Velocity{Linear: p.Velocity},
			physics.CollisionEventsEnabled{},
			missile,
			Stroke{Width: 2, Color: color.White, Antialias: true},
			DespawnAfter(10*time.Second),
			SmokeEmitter{Offset: Vec{X: -5}, Velocity: Vec{X: -1}.Mul(p.Velocity.Length() * 0.8), Timer: NewTimerWithFrequency(100.0)},
		).
		Observe(func(trigger On[TerrainContact], commands *Commands) {
			commands.Entity(trigger.Target).Despawn()
			commands.Trigger(Explode{Position: trigger.Event.Position, Radius: 20})
		}).
		Observe(func(trigger On[physics.OnCollisionStarted], commands *Commands) {
			commands.Entity(trigger.Target).Despawn()
			commands.Trigger(Explode{Position: trigger.Event.Position, Radius: 20})
		})

	commands.Spawn(
		AudioPlayerOf(assets.Audio("launch.ogg").Await()),
		PlaybackSettingsDespawn,
	)
}

func spawnSmokeSystem(
	commands *Commands,
	vt VirtualTime,
	query Query[struct {
		Transform  Transform
		SpawnSmoke *SmokeEmitter
		Velocity   Option[physics.Velocity]
	}],
) {
	for item := range query.Items() {
		item.SpawnSmoke.Timer.Tick(vt.Delta)

		rot := RotationMat(item.Transform.Rotation)

		for range item.SpawnSmoke.Timer.TimesFinishedThisTick() {
			r := rand.Float64()*5 + 2

			velocity := rot.Transform(item.SpawnSmoke.Velocity).
				Add(item.Velocity.OrZero().Linear).
				Add(Vec{
					X: rand.Float64() * 5,
					Y: rand.Float64() * 5,
				})

			lifetime := 500*time.Millisecond + time.Duration((rand.Float64()-0.5)*float64(100*time.Millisecond))

			// transform local offset into world space
			pos := item.Transform.AsAffine().Transform(item.SpawnSmoke.Offset)

			// add a small offset to the position
			pos = pos.Add(RandomVec[float64]().Mul(2.0))

			var puff Path
			puff.Circle(VecZero, r)

			commands.Spawn(
				puff,
				Fill{Color: color.RGBA(1, 1, 1, 0.2)},
				TransformFromXY(pos.XY()),
				DespawnAfter(lifetime),
				physics.Velocity{Linear: velocity},
			)
		}
	}
}
