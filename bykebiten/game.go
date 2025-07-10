package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
)

var TransformSystems = &byke.SystemSet{}

var GamePlugin byke.PluginFunc = func(app *byke.App) {
	app.InsertResource(WindowConfig{
		Title:  "Ebitengine",
		Width:  800,
		Height: 600,
	})

	app.InsertResource(MouseCursor{})
	app.InsertResource(RenderTarget{})
	app.InsertResource(ScreenSize{})

	app.InsertResource(MouseButtons{})
	app.InsertResource(Keys{})

	app.AddSystems(byke.First, updateMouseCursorSystem)

	app.AddSystems(byke.PreUpdate, byke.System(interactionSystem))

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleTransformSystem, propagateTransformSystem).
		Chain().
		InSet(TransformSystems))

	app.AddSystems(byke.PreRender,
		computeCachedVertices,
		computeSpriteSizeSystem,
		computeTextSizeSystem,
	)

	app.AddSystems(byke.Render, byke.System(renderSystem).Chain())

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

	var options ebiten.RunGameOptions
	options.SingleThread = true

	return ebiten.RunGameWithOptions(&game{World: world}, &options)
}

type game struct {
	World *byke.World

	screenSize gm.Vec
}

func (g *game) Update() error {
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.World.InsertResource(RenderTarget{Image: screen})
	g.World.InsertResource(ScreenSize{Vec: imageSizeOf(screen)})

	g.World.RunSchedule(byke.Main)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.screenSize = gm.Vec{X: float64(outsideWidth), Y: float64(outsideHeight)}
	return outsideWidth, outsideHeight
}

type ScreenSize struct {
	gm.Vec
}

type MouseCursor struct {
	gm.Vec
}

func updateMouseCursorSystem(cursor *MouseCursor) {
	x, y := ebiten.CursorPosition()
	cursor.X = float64(x)
	cursor.Y = float64(y)
}
