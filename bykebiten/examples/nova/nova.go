package main

import (
	"embed"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	. "github.com/oliverbestmann/byke/gm"
)

const SunRadius = 200.0

//go:embed assets
var assets embed.FS

func main() {
	var app App
	app.InsertResource(MakeAssetFS(assets))
	app.AddPlugin(GamePlugin)

	app.AddSystems(Startup, spawnCameraSystem, spawnSunSystem, spawnPlayerSystem, spawnObstaclesSystem)
	app.AddSystems(Update, System(movePlayerSystem, hitObstacleSystem).Chain())
	app.AddSystems(Update, System(emitThrustSystem).RunIf(TimerJustFinished(NewTimerWithFrequency(50))))

	app.MustRun()
}

type Player struct {
	Component[Player]
	DecentVelocity float64
	Thrust         bool
}

type Obstacle struct {
	Component[Obstacle]
	Radius float64
}

func spawnCameraSystem(commands *Commands) {
	commands.Spawn(
		TransformFromXY(SunRadius*1.1, 0.0),
		Camera{},
		OrthographicProjection{
			ViewportOrigin: VecSplat(0.5),
			ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 200.0},
			Scale:          0.25,
		},
	)
}

func spawnSunSystem(commands *Commands, assets *Assets) {
	var inputs ShaderInput
	inputs.Put("Inner", color.RGBA(3.968, 0.372, 0.051, 1.0))
	inputs.Put("Outer", color.RGBA(2.868, 0.602, 0.061, 1.0))
	inputs.Put("BlurStart", 0.5)

	commands.Spawn(
		Circle(SunRadius/0.5, 48),
		assets.Shader("sun.kage").Await(),
		inputs,
		LayerOf(10),
	)
}

func spawnPlayerSystem(commands *Commands) {
	commands.Spawn(
		Player{},
		TransformFromXY(SunRadius*1.1, 0),

		Visible,

		SpawnChild(
			LayerOf(9),
			RegularPolygon(1, 3),
			NewTransform().
				WithScale(1, 1.5).
				WithRotation(DegToRad(-90)),
		),
	)
}

func spawnObstaclesSystem(commands *Commands) {
	for range 100 {
		pos := Vec{X: RandomIn(SunRadius*1.05, SunRadius*1.3)}.Rotated(RandomAngle())

		radius := RandomIn(0.5, 1.5)

		var points []Vec
		var angle Rad
		for angle < 2*math.Pi {
			point := Vec{X: radius + RandomIn(-0.3, 0.5)}.Rotated(angle)
			points = append(points, point)

			step := RandomIn(DegToRad(20), DegToRad(40))
			angle += step
		}

		commands.Spawn(
			Obstacle{Radius: radius},
			TransformFromXY(pos.XY()),
			ColorTint{Color: color.RGBA(1.0, 0.3, 0.08, 1.0)},
			Polygon(points),
		)
	}
}

func movePlayerSystem(
	vt VirtualTime,
	keys Keys,
	playerQuery Query[struct {
	Player    *Player
	Transform *Transform
}],
	cameraQuery Query[struct {
	_          With[Camera]
	Transform  *Transform
	Projection *OrthographicProjection
}],

) {
	player, ok := playerQuery.Single()
	if !ok {
		return
	}

	camera, _ := cameraQuery.Single()

	pos := player.Transform.Translation

	// gravity
	player.Player.DecentVelocity += 10 * vt.DeltaSecs

	// TODO move into player input/control system
	// player input
	player.Player.Thrust = keys.IsPressed(ebiten.KeySpace)
	if player.Player.Thrust {
		player.Player.DecentVelocity -= 30 * vt.DeltaSecs
	}

	dist := pos.Length() - player.Player.DecentVelocity*vt.DeltaSecs
	if dist < SunRadius {
		// limit to going up
		player.Player.DecentVelocity = min(0, player.Player.DecentVelocity)
		dist = SunRadius
	}

	// move in the direction around the sun
	dir := Vec{X: pos.Y, Y: -pos.X}.Normalized().Mul(-35.0)
	posNew := pos.Add(dir.Mul(vt.DeltaSecs)).Normalized().Mul(dist)

	player.Transform.Translation = posNew
	player.Transform.Rotation = pos.AngleTo(posNew)

	// TODO move into camera follow system
	camera.Transform.Translation = posNew
}

func emitThrustSystem(commands *Commands, player Single[struct {
	Player    Player
	Transform Transform
}]) {
	p := player.Value
	if !p.Player.Thrust {
		return
	}

	commands.Spawn(
		Rectangle(Vec{X: 0.5, Y: 0.5}.Mul(RandomIn(1.0, 2))),
		DespawnAfter(time.Duration(RandomIn(150, 200))*time.Millisecond),
		ColorTint{Color: color.White.WithAlpha(0.5)},
		p.Transform.WithRotation(Rad(rand.Float64())*2*math.Pi),
	)
}

func hitObstacleSystem(
	commands *Commands,

	player Single[struct {
	_         With[Player]
	Transform Transform
}],
	obstacles Query[struct {
	EntityId
	Obstacle  Obstacle
	Transform Transform
}],
) {
	p := player.Value

	for item := range obstacles.Items() {
		dist := item.Transform.Translation.DistanceTo(p.Transform.Translation)

		if dist < 1+item.Obstacle.Radius {
			commands.Entity(item.EntityId).Despawn()
		}
	}
}
