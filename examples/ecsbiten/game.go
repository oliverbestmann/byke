package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	ecs "gobevy"
	"time"
)

var Startup = &ecs.Schedule{}

var First = &ecs.Schedule{}
var PreUpdate = &ecs.Schedule{}
var Update = &ecs.Schedule{}
var PostUpdate = &ecs.Schedule{}
var PreRender = &ecs.Schedule{}
var Render = &ecs.Schedule{}
var PostRender = &ecs.Schedule{}
var Last = &ecs.Schedule{}

func Plugin(app *ecs.App) {
	app.InsertResource(WindowConfig{
		Title:  "Ebitengine",
		Width:  800,
		Height: 600,
	})

	app.InsertResource(VirtualTime{Scale: 1})
	app.InsertResource(MouseCursor{})

	app.AddSystems(First, updateVirtualTime, updateMouseCursor)
	app.AddSystems(Render, renderSpritesSystem)

	// start the game
	app.AddSystems(ecs.RunWorld, runGameSystem)
}

type WindowConfig struct {
	Title  string
	Width  int
	Height int
}

func runGameSystem(world *ecs.World, win WindowConfig) {
	ebiten.SetWindowTitle(win.Title)
	ebiten.SetWindowSize(win.Width, win.Height)

	_ = ebiten.RunGame(&Game{World: world})
}

type Game struct {
	World *ecs.World

	initialized bool
	screenSize  Vec
}

func (g *Game) Init() {
	g.World.RunSchedule(Startup)
}

func (g *Game) Update() error {
	g.World.InsertResource(ScreenSize(g.screenSize))

	if !g.initialized {
		g.initialized = true
		g.Init()
	}

	// start the new frame
	g.World.RunSchedule(First)

	// the update schedule
	g.World.RunSchedule(PreUpdate)
	g.World.RunSchedule(Update)
	g.World.RunSchedule(PostUpdate)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.World.InsertResource(screen)

	g.World.RunSchedule(PreRender)
	g.World.RunSchedule(Render)
	g.World.RunSchedule(PostRender)

	// end the frame
	g.World.RunSchedule(Last)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.screenSize = Vec{X: float64(outsideWidth), Y: float64(outsideHeight)}
	return outsideWidth, outsideHeight
}

type ScreenSize Vec

type MouseCursor Vec

type VirtualTime struct {
	Elapsed   time.Duration
	Delta     time.Duration
	DeltaSecs float64

	Scale float64

	// the time of the last update
	updateTime time.Time
}

func updateVirtualTime(v *VirtualTime) {
	now := time.Now()

	if v.updateTime.IsZero() {
		v.updateTime = now
		return
	}

	v.Delta = time.Duration(float64(now.Sub(v.updateTime)) * v.Scale)
	v.DeltaSecs = v.Delta.Seconds()
	v.Elapsed += v.Delta

	v.updateTime = now
}

func updateMouseCursor(cursor *MouseCursor) {
	x, y := ebiten.CursorPosition()
	cursor.X = float64(x)
	cursor.Y = float64(y)
}
