package main

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	. "github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})

	slog.SetDefault(slog.New(handler))
}

const PaddleWidth = 5
const PaddleHeight = 50
const PaddleAbsX = 450
const BallRadius = 10

const AreaHeight = 500
const AreaWidth = 1000

const PaddleSpeed = 300
const BallSpeed = 500

var InputSystems = &SystemSet{Name: "InputSystem"}
var MovementSystems = &SystemSet{Name: "MovementSystems"}

type GameState int

const GameStateWaiting GameState = 0
const GameStatePlaying GameState = 1
const GameStateLost GameState = 2

func main() {
	var app App

	// update fixed simulation with ~128hz
	app.InsertResource(FixedTime{StepInterval: 1 * time.Second / 128})

	app.InitState(StateType[GameState]{InitialValue: GameStatePlaying})

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)
	app.AddSystems(Startup, setupSystem)

	app.AddSystems(Update, System(handleInputSystem).InSet(InputSystems))

	app.AddSystems(FixedUpdate, System(paddleCollisionSystem, restrictBallVerticalSystem, applyVelocitySystems, registerGameLostSystem).
		Chain().
		RunIf(InState(GameStatePlaying)).
		InSet(MovementSystems),
	)
	app.AddSystems(Update, paddleGlowSystem)

	app.AddSystems(OnEnter(GameStateLost), enterLostStateSystem)

	app.ConfigureSystemSets(Update, ChainSystemSets(InputSystems, MovementSystems))

	// TODO does not work yet
	// configure all movement systems to only run while the game is on
	// MovementSystems.RunIf(InState(GameStatePlaying))

	app.MustRun()
}

func setupSystem(commands *Commands) {
	// The game area is 1000x500 px large
	commands.Spawn(
		Camera{},
		ClearColor{Color: ColorBlack},
		HDR{},
		TransformFromXYZ(0, 0, -1.0),
		OrthographicProjection{
			ViewportOrigin: Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeAutoMin{MinWidth: 1000 + 100, MinHeight: 500 + 50},
		},
	)

	// rectangle to highlight the game area
	commands.Spawn(
		Mesh2d{Mesh: Rectangle(Vec2f{1000, 500})},
		ColorMaterial{Tint: ColorSRGBA(0.1, 0.1, 0.1, 1.0)},
	)

	// the ball
	commands.Spawn(
		TransformFromXYZ(0, 0, 0.1),
		Ball{},
		Velocity{Value: Vec2f{2, rand.Float32() * 0.01}.Normalize().Scale(BallSpeed)},
		Mesh2d{Mesh: Circle(BallRadius, 32)},
		ColorMaterial{Tint: ColorSRGBA(0, 5, 0, 1.0)},
	)

	// left paddle
	commands.Spawn(
		TransformFromXYZ(-PaddleAbsX, 0, 0.2),
		Paddle{IsLeft: true},
		PlayerPaddle{},
		Mesh2d{Mesh: Rectangle(Vec2f{PaddleWidth, PaddleHeight})},
		ColorMaterial{Tint: ColorSRGBA(10, 0, 5, 1.0)},
		Velocity{},
	)

	// right paddle
	commands.Spawn(
		TransformFromXYZ(PaddleAbsX, 0, 0.2),
		Paddle{},
		Mesh2d{Mesh: Rectangle(Vec2f{PaddleWidth, PaddleHeight})},
		ColorMaterial{Tint: ColorSRGBA(5, 0, 10, 1.0)},
		Velocity{},
	)
}

type Ball struct {
	Component[Ball]
}

type Velocity struct {
	Component[Velocity]
	Value Vec2f
}

type Paddle struct {
	ImmutableComponent[Paddle]
	IsLeft bool
}

type PlayerPaddle struct {
	ImmutableComponent[PlayerPaddle]
}

func applyVelocitySystems(ft FixedTime, query Query[struct {
	Transform *Transform
	Velocity  Velocity
}]) {
	for item := range query.Items() {
		delta := item.Velocity.Value.Scale(ft.DeltaSecs)
		item.Transform.Translation = item.Transform.Translation.Add(delta.Extend(0.0))
	}
}

func paddleCollisionSystem(
	ballsQuery Query[struct {
		_         With[Ball]
		Transform Transform
		Velocity  *Velocity
	}],
	paddlesQuery Query[struct {
		Paddle    Paddle
		Transform Transform
		Velocity  Velocity
	}],
) {
	for ball := range ballsQuery.Items() {
		ballX := ball.Transform.Translation[0]

		if math.Abs(float64(ballX)) < PaddleAbsX-PaddleWidth/2-BallRadius {
			// not near a paddle
			continue
		}

		isLeft := ballX < 0

		for paddle := range paddlesQuery.Items() {
			if paddle.Paddle.IsLeft != isLeft {
				continue
			}

			ballY := ball.Transform.Translation[1]

			paddleY := paddle.Transform.Translation[1]
			paddleMin := paddleY - PaddleHeight/2 - BallRadius
			paddleMax := paddleY + PaddleHeight/2 + BallRadius

			if ballY < paddleMin || ballY > paddleMax {
				continue
			}

			ball.Velocity.Value[0] *= -1
			ball.Velocity.Value[1] += paddle.Velocity.Value[1] * rand.Float32()
		}
	}
}

func paddleGlowSystem(
	ballsQuery Query[struct {
		_         With[Ball]
		Transform Transform
	}],
	paddlesQuery Query[struct {
		_         With[Paddle]
		Transform Transform
		Material  *ColorMaterial
	}],
) {
	for ball := range ballsQuery.Items() {
		for paddle := range paddlesQuery.Items() {
			dist := max(50, ball.Transform.Translation.Sub(paddle.Transform.Translation).Length()-50)

			nColor := paddle.Material.Tint.ToVec().Truncate().Normalize()
			col := nColor.Scale(10).Add(nColor.Scale(5000 / dist))
			paddle.Material.Tint = ColorOf(col.Extend(1.0))
		}
	}
}

func restrictBallVerticalSystem(
	ballsQuery Query[struct {
		_         With[Ball]
		Transform *Transform
		Velocity  *Velocity
	}],
) {
	for ball := range ballsQuery.Items() {
		ballY := ball.Transform.Translation[1]

		if ballY-BallRadius <= -AreaHeight/2 {
			ball.Velocity.Value[1] *= -1
		}

		if ballY+BallRadius >= AreaHeight/2 {
			ball.Velocity.Value[1] *= -1
		}
	}
}

func handleInputSystem(
	keys Keys,
	paddle Single[struct {
		_        With[Paddle]
		_        With[PlayerPaddle]
		Velocity *Velocity
	}],
) {
	var direction float32

	if keys.IsPressed(vyn.KeyArrowUp) {
		direction = 1
	}

	if keys.IsPressed(vyn.KeyArrowDown) {
		direction = -1
	}

	paddle.Get().Velocity.Value[1] = PaddleSpeed * direction
}

func registerGameLostSystem(
	gameState *NextState[GameState],

	ballsQuery Query[struct {
		_         With[Ball]
		Transform Transform
	}],
) {
	for ball := range ballsQuery.Items() {
		ballX := ball.Transform.Translation[0]
		if math.Abs(float64(ballX)) > AreaWidth/2+BallRadius {
			gameState.Set(GameStateLost)
		}
	}
}

func enterLostStateSystem(
	commands *Commands,
) {
	commands.Spawn(
		TransformFromXYZ(0, 0, 0.5),
		Text{
			Text:  "You lose!",
			Color: ColorSRGBA(10, 0, 0, 1),
			Size:  48,
		},
	)
}
