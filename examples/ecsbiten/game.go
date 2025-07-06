package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
)

var Plugin byke.PluginFunc = func(app *byke.App) {
	app.InsertResource(WindowConfig{
		Title:  "Ebitengine",
		Width:  800,
		Height: 600,
	})

	app.InsertResource(MouseCursor{})
	app.InsertResource(RenderTarget{})
	app.InsertResource(ScreenSize{})

	app.InsertResource(Keys{})

	app.AddSystems(byke.First, updateMouseCursor)
	app.AddSystems(byke.Render, renderSpritesSystem)

	// start the game
	app.RunWorld(runWorld)
}

type WindowConfig struct {
	Title  string
	Width  int
	Height int
}

func runWorld(world *byke.World) error {
	win, _ := byke.ResourceOf[WindowConfig](world)
	ebiten.SetWindowTitle(win.Title)
	ebiten.SetWindowSize(win.Width, win.Height)

	return ebiten.RunGame(&game{World: world})
}

type game struct {
	World *byke.World

	screenSize Vec
}

func (g *game) Update() error {
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.World.InsertResource(RenderTarget{Image: screen})
	g.World.InsertResource(ScreenSize{Vec: ImageSizeOf(screen)})

	g.World.RunSchedule(byke.Main)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.screenSize = Vec{X: float64(outsideWidth), Y: float64(outsideHeight)}
	return outsideWidth, outsideHeight
}

type ScreenSize struct {
	Vec
}

type MouseCursor struct {
	Vec
}

func updateMouseCursor(cursor *MouseCursor) {
	x, y := ebiten.CursorPosition()
	cursor.X = float64(x)
	cursor.Y = float64(y)
}
