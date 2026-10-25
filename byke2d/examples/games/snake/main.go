package main

import (
	"math/rand/v2"
	"time"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// Snake on a 20x15 grid. The body never moves: every step spawns a new segment at
// the head, and each segment despawns after as many steps as the snake is long.

const (
	cellSize = 32
	cols     = 20
	rows     = 15
)

type Cell = [2]int

// Snake holds the state of the head.
type Snake struct {
	Head    Cell
	Dir     Cell
	NextDir Cell
	Length  int
}

var newSnake = Snake{Head: Cell{5, 7}, Dir: Cell{1, 0}, NextDir: Cell{1, 0}, Length: 3}

// Segment is a part of the body. It lives for the given number of steps.
type Segment struct {
	Component[Segment]
	Cell  Cell
	Steps int
}

type Food struct {
	Component[Food]
	Cell Cell
}

// fixed seed, so every run looks the same
var rng = rand.New(rand.NewPCG(1, 2))

func main() {
	var app App

	// the snake moves ten cells per second
	app.InsertResource(FixedTime{StepInterval: time.Second / 10})
	app.InsertResource(newSnake)

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Arrow keys to steer"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, steerSystem)
	app.AddSystems(FixedUpdate, System(stepSystem, eatSystem).Chain())

	shared.RunAppInTest(app, shared.FramesToSnapshot{10, 60})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{
			ScalingMode: ScalingModeAutoMin{MinWidth: cols * cellSize, MinHeight: rows * cellSize},
		},
	)

	// the grid
	commands.Spawn(
		TransformFromXYZ(cols*cellSize/2, rows*cellSize/2, -1),
		MeshInstance{Mesh: Rectangle(glm.Vec2f{cols * cellSize, rows * cellSize})},
		ColorMaterial{Color: ColorSRGB(0.1, 0.1, 0.15)},
	)

	commands.Spawn(
		Food{Cell: Cell{12, 7}},
		TransformAt(Cell{12, 7}),
		MeshInstance{Mesh: Circle(cellSize/2-4, 24)},
		ColorMaterial{Color: ColorSRGB(1, 0.3, 0.3)},
	)
}

// TransformAt places an entity in the center of a cell.
func TransformAt(cell Cell) Transform {
	return TransformFromXY((float32(cell[0])+0.5)*cellSize, (float32(cell[1])+0.5)*cellSize)
}

func steerSystem(keys Keys, snake *Snake) {
	dirs := map[vyn.Key]Cell{
		vyn.KeyArrowLeft:  {-1, 0},
		vyn.KeyArrowRight: {1, 0},
		vyn.KeyArrowUp:    {0, 1},
		vyn.KeyArrowDown:  {0, -1},
	}

	for key, dir := range dirs {
		// the snake can not turn back into itself
		if keys.IsJustPressed(key) && dir != (Cell{-snake.Dir[0], -snake.Dir[1]}) {
			snake.NextDir = dir
		}
	}
}

func stepSystem(commands *Commands, snake *Snake, segments Query[struct {
	EntityId
	Segment *Segment
}],
) {
	snake.Dir = snake.NextDir

	// move the head, leaving the grid on one side enters it on the other
	snake.Head = Cell{
		(snake.Head[0] + snake.Dir[0] + cols) % cols,
		(snake.Head[1] + snake.Dir[1] + rows) % rows,
	}

	bitten := false
	for item := range segments.Items() {
		bitten = bitten || item.Segment.Cell == snake.Head
	}

	// on a bite the whole body goes away and the snake starts over
	for item := range segments.Items() {
		item.Segment.Steps--
		if bitten || item.Segment.Steps <= 0 {
			commands.Entity(item.EntityId).Despawn()
		}
	}

	if bitten {
		*snake = newSnake
		return
	}

	commands.Spawn(
		Segment{Cell: snake.Head, Steps: snake.Length},
		TransformAt(snake.Head),
		MeshInstance{Mesh: Rectangle(glm.Vec2f{cellSize - 2, cellSize - 2})},
		ColorMaterial{Color: ColorSRGB(0.4, 0.9, 0.4)},
	)
}

func eatSystem(snake *Snake, food Single[struct {
	Food      *Food
	Transform *Transform
}],
) {
	if food.Value.Food.Cell != snake.Head {
		return
	}

	snake.Length++

	// shortcut: food can land under the body, where it stays hidden until the body moves away
	cell := Cell{rng.IntN(cols), rng.IntN(rows)}
	food.Value.Food.Cell = cell
	*food.Value.Transform = TransformAt(cell)
}
