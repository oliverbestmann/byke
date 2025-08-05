package bykebiten

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
)

var TransformSystems = &byke.SystemSet{}
var VisibilitySystems = &byke.SystemSet{}
var AudioSystems = &byke.SystemSet{}

type AssetFS struct {
	fs.FS
}

func MakeAssetFS(root fs.FS) AssetFS {
	sub, _ := fs.Sub(root, "assets")
	return AssetFS{FS: sub}
}

func GamePlugin(app *byke.App) {
	assetFs, ok := byke.ResourceOf[AssetFS](app.World())
	if !ok {
		assetFs = &AssetFS{FS: os.DirFS("assets")}
	}

	app.InsertResource(WindowConfig{
		Title:  "Ebitengine",
		Width:  800,
		Height: 600,
	})

	app.InsertResource(MouseCursor{})
	app.InsertResource(screenRenderTarget{})
	app.InsertResource(ScreenSize{})
	app.InsertResource(DrawFinalScreen(ebiten.DefaultDrawFinalScreen))

	app.InsertResource(MouseButtons{})
	app.InsertResource(Keys{})

	app.InsertResource(AudioContext{audioContext})
	app.InsertResource(GlobalVolume{Volume: 1.0})
	app.InsertResource(GlobalSpatialScale{Scale: gm.VecOne})

	assets := makeAssets(assetFs.FS, ImageLoader{}, AudioLoader{})
	app.InsertResource(assets)

	app.AddEvent(byke.EventType[AppExit]())

	app.AddSystems(byke.First, updateMouseCursorSystem)

	app.AddSystems(byke.PreUpdate, byke.System(interactionSystem))

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleTransformSystem, propagateTransformSystem).
		Chain().
		InSet(TransformSystems))

	app.AddSystems(byke.PostUpdate, byke.
		System(syncSimpleVisibilitySystem, propagateVisibilitySystem).
		Chain().
		InSet(VisibilitySystems))

	app.AddSystems(byke.PreRender,
		byke.System(updateTileCache).Before(computeSpriteSizeSystem),
		updateUniformsSystem,
	)

	app.AddSystems(byke.PreRender,
		computePathSizeSystem,
		computeSpriteSizeSystem,
		computeTextSizeSystem,
		computeMeshSizeSystem,
	)

	app.AddSystems(byke.PreRender,
		byke.System(createAudioSinkSystem, adjustSpatialAudioVolume, cleanupAudioSinkSystem).Chain().InSet(AudioSystems))

	app.AddSystems(byke.Render, byke.System(renderSystem).Chain())

	app.AddSystems(byke.Update, byke.
		System(toggleRenderTimingsSystem).
		RunIf(KeyJustPressed(ebiten.KeyD)))

	app.AddSystems(byke.PostRender, byke.
		System(renderTimingsSystem).
		RunIf(byke.ResourceExists[byke.TimingStats]))

	// read AppExit events last so the next update tick can already exit the app.
	app.AddSystems(byke.Last, readAppExitEventsSystem)

	// start the game
	app.RunWorld(runWorld)
}

func toggleRenderTimingsSystem(world *byke.World) {
	_, ok := byke.ResourceOf[byke.TimingStats](world)
	if ok {
		world.RemoveResource(reflect.TypeFor[byke.TimingStats]())
	} else {
		world.InsertResource(byke.NewTimingStats())
	}
}

func renderTimingsSystem(
	timings byke.TimingStats,
	renderTarget screenRenderTarget,
	frameCounter *byke.Local[int],
	image *byke.Local[*ebiten.Image],
) {
	frameCounter.Value += 1
	if frameCounter.Value%30 != 0 && image.Value != nil {
		renderTarget.Image.DrawImage(image.Value, nil)
		return
	}

	if image.Value == nil || image.Value.Bounds() != renderTarget.Image.Bounds() {
		b := renderTarget.Image.Bounds()
		image.Value = ebiten.NewImage(b.Dx(), b.Dy())
	}

	image.Value.Clear()

	var row int

	var maxNameLength int

	for _, scheduleId := range timings.ScheduleOrder {
		maxNameLength = max(maxNameLength, len(scheduleId.String()))
	}

	for _, scheduleId := range timings.ScheduleOrder {
		t := timings.BySchedule[scheduleId]

		text := fmt.Sprintf("%-[1]*s runs=%5d, latest=%4.2fms, min=%4.2fms, max=%4.2fms, avg=%4.2fms",
			maxNameLength,
			scheduleId,
			t.Count,
			t.Latest.Seconds()*1000,
			t.Min.Seconds()*1000,
			t.Max.Seconds()*1000,
			t.MovingAverage.Seconds()*1000,
		)

		ebitenutil.DebugPrintAt(image.Value, text, 16, 16+16*row)
		row += 1
	}

	type System struct {
		Name    string
		Timings byke.Timings
	}

	var systems []System
	for sys, t := range timings.BySystem {
		if t.MovingAverage < 250*time.Microsecond {
			continue
		}

		systems = append(systems, System{sys.Name, t})
		maxNameLength = max(maxNameLength, len(sys.Name))
	}

	slices.SortFunc(systems, func(a, b System) int {
		return int(b.Timings.MovingAverage - a.Timings.MovingAverage)
	})

	row += 1

	for _, sys := range systems {
		text := fmt.Sprintf("%-[1]*s runs=%5d, latest:%6.2fms, min:%6.2fms, max:%6.2fms, avg:%6.2fms",
			maxNameLength,
			sys.Name,
			sys.Timings.Count,
			sys.Timings.Latest.Seconds()*1000,
			sys.Timings.Min.Seconds()*1000,
			sys.Timings.Max.Seconds()*1000,
			sys.Timings.MovingAverage.Seconds()*1000,
		)

		ebitenutil.DebugPrintAt(image.Value, text, 16, 16+16*row)
		row += 1
	}

	// draw the now cached text
	renderTarget.Image.DrawImage(image.Value, nil)
}

type WindowConfig struct {
	Title         string
	Width         int
	Height        int
	DisableResize bool
}

func runWorld(world *byke.World) error {
	world.InsertResource(game{World: world})

	win, _ := byke.ResourceOf[WindowConfig](world)
	ebiten.SetWindowTitle(win.Title)
	ebiten.SetWindowSize(win.Width, win.Height)

	if !win.DisableResize {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}

	var options ebiten.RunGameOptions
	options.SingleThread = true

	theGame, _ := byke.ResourceOf[game](world)
	return ebiten.RunGameWithOptions(theGame, &options)
}

type game struct {
	World *byke.World

	// set to a non nil value to exit the app
	appExit    error
	screenSize gm.Vec
}

func (g *game) Update() error {
	return g.appExit
}

func (g *game) Draw(screen *ebiten.Image) {
	g.World.InsertResource(screenRenderTarget{Image: screen})
	g.World.InsertResource(ScreenSize{Vec: imageSizeOf(screen)})

	g.World.RunSchedule(byke.Main)
}

func (g *game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	draw, ok := byke.ResourceOf[DrawFinalScreen](g.World)

	if ok {
		(*draw)(screen, offscreen, geoM)
	} else {
		ebiten.DefaultDrawFinalScreen(screen, offscreen, geoM)
	}
}

func (g *game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.screenSize = gm.Vec{X: float64(outsideWidth), Y: float64(outsideHeight)}
	return outsideWidth, outsideHeight
}

type ScreenSize struct {
	gm.Vec
}

// DrawFinalScreen is a resource that overrides the default DrawFinalScreen implementation
type DrawFinalScreen func(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM)

type MouseCursor struct {
	gm.Vec
}

func updateMouseCursorSystem(cursor *MouseCursor) {
	x, y := ebiten.CursorPosition()
	cursor.X = float64(x)
	cursor.Y = float64(y)
}

func readAppExitEventsSystem(events *byke.EventReader[AppExit], game *game) {
	for _, ev := range events.Read() {
		game.appExit = ev.error
	}
}
