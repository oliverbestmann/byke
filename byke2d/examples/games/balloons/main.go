package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// Pop the balloons before they float away. After three missed balloons the game is over.
// The two game states decide which systems run, and entities of a state despawn
// when it ends.

type GameState int

const (
	Playing GameState = iota
	GameOver
)

type Score struct {
	Popped int
	Missed int
}

type Balloon struct {
	ImmutableComponent[Balloon]
}

type Velocity struct {
	Component[Velocity]
	glm.Vec2f
}

type SpawnTimer struct {
	Timer
}

type ScoreText struct {
	ImmutableComponent[ScoreText]
}

const balloonRadius = 28

var colors = []Color{
	ColorSRGB(1, 0.3, 0.3),
	ColorSRGB(1, 0.8, 0.2),
	ColorSRGB(0.3, 0.8, 1),
	ColorSRGB(0.6, 0.4, 1),
	ColorSRGB(0.4, 1, 0.5),
}

// fixed seed, so every run looks the same
var rng = rand.New(rand.NewPCG(1, 2))

func main() {
	var app App

	app.InitState(Playing)
	app.InsertResource(Score{})
	app.InsertResource(SpawnTimer{NewTimer(600*time.Millisecond, TimerModeRepeating)})

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Click the balloons"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, moveSystem, shrinkSystem, scoreTextSystem)

	app.AddSystems(OnEnter(Playing), func(score *Score) { *score = Score{} })
	app.AddSystems(Update, System(spawnBalloonSystem, popSystem, escapeSystem).RunIf(InState(Playing)))

	app.AddSystems(OnEnter(GameOver), gameOverSystem)
	app.AddSystems(Update, System(restartSystem).
		RunIf(InState(GameOver)).
		RunIf(MouseButtonIsJustPressed(vyn.MouseButtonLeft)))

	shared.RunAppInTest(app, shared.FramesToSnapshot{60, 300})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{ViewportOrigin: glm.Vec2f{0.5, 0.5}},
	)

	// the score sits large in the background
	commands.Spawn(
		ScoreText{},
		TransformFromXYZ(0, 0, -1),
		AnchorCenter,
		Text{Size: 160, Color: ColorSRGB(0.3, 0.3, 0.4)},
	)
}

func spawnBalloonSystem(commands *Commands, vt VirtualTime, screen ScreenSize, score Score, timer *SpawnTimer) {
	if !timer.Tick(vt.Delta).JustFinished() {
		return
	}

	// balloons get faster with every pop
	speed := 100 + rng.Float32()*60 + float32(score.Popped)*5

	x := (rng.Float32() - 0.5) * (screen.Vec2f[0] - 2*balloonRadius)
	y := -screen.Vec2f[1]/2 - balloonRadius

	commands.Spawn(
		Balloon{},
		Velocity{Vec2f: glm.Vec2f{0, speed}},
		TransformFromXY(x, y),
		MeshInstance{Mesh: Ellipse(glm.Vec2f{2 * balloonRadius, 2.4 * balloonRadius}, 32)},
		ColorMaterial{Color: colors[rng.IntN(len(colors))]},
		DespawnOnExitState(Playing),
	)
}

func moveSystem(vt VirtualTime, query Query[struct {
	Velocity  Velocity
	Transform *Transform
}],
) {
	for item := range query.Items() {
		step := item.Velocity.Scale(vt.DeltaSecs)
		item.Transform.Translation = item.Transform.Translation.Add(step.Extend(0))
	}
}

func popSystem(
	commands *Commands,
	buttons MouseButtons,
	cursor MouseCursor,
	screen ScreenSize,
	score *Score,
	balloons Query[struct {
		_ With[Balloon]
		EntityId
		Transform Transform
		Material  ColorMaterial
	}],
) {
	if !buttons.IsJustPressed(vyn.MouseButtonLeft) {
		return
	}

	// the cursor is in window pixels with y down, the camera is centered with y up
	click := glm.Vec2f{cursor.Vec2f[0] - screen.Vec2f[0]/2, screen.Vec2f[1]/2 - cursor.Vec2f[1]}

	for balloon := range balloons.Items() {
		pos := balloon.Transform.Translation.Truncate()
		if pos.Sub(click).Length() > balloonRadius {
			continue
		}

		commands.Entity(balloon.EntityId).Despawn()
		score.Popped++

		// burst into small pieces that fly apart and shrink
		for idx := range 10 {
			angle := float64(idx) / 10 * 2 * math.Pi
			dir := glm.Vec2f{float32(math.Cos(angle)), float32(math.Sin(angle))}

			commands.Spawn(
				Velocity{Vec2f: dir.Scale(250)},
				TransformFromXYZ(pos[0], pos[1], 1),
				MeshInstance{Mesh: Circle(6, 12)},
				ColorMaterial{Color: balloon.Material.Color},
				DespawnAfter(400*time.Millisecond),
			)
		}

		// pop only one balloon per click
		return
	}
}

// The despawn timer also drives the size of each piece.
func shrinkSystem(query Query[struct {
	Despawn   DespawnWithDelay
	Transform *Transform
}],
) {
	for item := range query.Items() {
		s := float32(item.Despawn.Timer.FractionRemaining())
		item.Transform.Scale = glm.Vec3f{s, s, 1}
	}
}

func escapeSystem(
	commands *Commands,
	screen ScreenSize,
	score *Score,
	next *NextState[GameState],
	balloons Query[struct {
		_ With[Balloon]
		EntityId
		Transform Transform
	}],
) {
	for balloon := range balloons.Items() {
		if balloon.Transform.Translation[1] < screen.Vec2f[1]/2+balloonRadius {
			continue
		}

		commands.Entity(balloon.EntityId).Despawn()
		score.Missed++

		if score.Missed >= 3 {
			next.Set(GameOver)
		}
	}
}

func gameOverSystem(commands *Commands) {
	commands.Spawn(
		TransformFromXYZ(0, -100, 2),
		AnchorCenter,
		Text{Text: "Game over, click to play again", Size: 32, Color: ColorWhite},
		DespawnOnExitState(GameOver),
	)
}

func restartSystem(next *NextState[GameState]) {
	next.Set(Playing)
}

func scoreTextSystem(score Score, text Single[struct {
	_    With[ScoreText]
	Text *Text
}],
) {
	value := fmt.Sprint(score.Popped)
	if text.Value.Text.Text != value {
		text.Value.Text.Text = value
	}
}
