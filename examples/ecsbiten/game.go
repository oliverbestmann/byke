package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"time"
)

var PreStartup = byke.PreStartup
var Startup = byke.Startup
var PostStartup = byke.PostStartup

var First = byke.First
var PreUpdate = byke.PreUpdate
var StateTransition = byke.StateTransition
var Update = byke.Update
var PostUpdate = byke.PostUpdate
var PreRender = byke.PreRender
var Render = byke.Render
var PostRender = byke.PostRender
var Last = byke.Last

var Plugin byke.PluginFunc = func(app *byke.App) {
	app.InsertResource(WindowConfig{
		Title:  "Ebitengine",
		Width:  800,
		Height: 600,
	})

	app.InsertResource(VirtualTime{Scale: 1})
	app.InsertResource(MouseCursor{})
	app.InsertResource(RenderTarget{})
	app.InsertResource(ScreenSize{})

	app.InsertResource(Keys{})

	app.AddSystems(First, updateVirtualTime, updateMouseCursor)
	app.AddSystems(Render, renderSpritesSystem)

	// start the game
	app.RunWorld(runWorld)
}

type WindowConfig struct {
	Title  string
	Width  int
	Height int
}

func runWorld(world *byke.World) error {
	world.RunSystem(func(win WindowConfig) {
		ebiten.SetWindowTitle(win.Title)
		ebiten.SetWindowSize(win.Width, win.Height)
	})

	return ebiten.RunGame(&game{World: world})
}

type game struct {
	World *byke.World

	initialized bool
	screenSize  Vec
}

func (g *game) Init() {
	g.World.RunSchedule(PreStartup)
	g.World.RunSchedule(StateTransition)
	g.World.RunSchedule(Startup)
	g.World.RunSchedule(PostStartup)
}

func (g *game) Update() error {
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.World.InsertResource(RenderTarget{Image: screen})
	g.World.InsertResource(ScreenSize{Vec: ImageSizeOf(screen)})

	if !g.initialized {
		g.initialized = true
		g.Init()
	}

	// start the new frame
	g.World.RunSchedule(First)

	// the update schedule
	g.World.RunSchedule(PreUpdate)
	g.World.RunSchedule(StateTransition)
	g.World.RunSchedule(Update)
	g.World.RunSchedule(PostUpdate)

	g.World.RunSchedule(PreRender)
	g.World.RunSchedule(Render)
	g.World.RunSchedule(PostRender)

	// end the frame
	g.World.RunSchedule(Last)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.screenSize = Vec{X: float64(outsideWidth), Y: float64(outsideHeight)}
	return outsideWidth, outsideHeight
}

type ScreenSize struct {
	Vec
}

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
