package byke2d

import (
	"log/slog"
	"math"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

var _ = byke.ValidateComponent[DebugCamera]()

type DebugCamera struct {
	byke.ImmutableComponent[DebugCamera]
}

type DebugState bool

const (
	DebugStateOff DebugState = false
	DebugStateOn  DebugState = true
)

type debugMetricsTextMaker struct {
	byke.ImmutableComponent[debugMetricsTextMaker]
}

func pluginDebug(app *byke.App) {
	app.AddSystems(byke.Update, byke.System(dumpTree).
		RunIf(KeyIsJustPressed(vyn.KeyT)).
		RunIf(KeyIsPressed(vyn.KeyShiftLeft)))

	app.AddSystems(byke.Update, byke.System(printRenderContextMetricsSystem).
		RunIf(KeyIsJustPressed(vyn.KeyM)).
		RunIf(KeyIsPressed(vyn.KeyShiftLeft)))

	app.AddSystems(byke.Update, byke.System(printCameraGlobalPositionSystem).
		RunIf(KeyIsJustPressed(vyn.KeyP)).
		RunIf(KeyIsPressed(vyn.KeyShiftLeft)))

	app.AddSystems(byke.Update, byke.System(toggleDebugStateSystem).
		RunIf(KeyIsJustPressed(vyn.KeyD)).
		RunIf(KeyIsPressed(vyn.KeyShiftLeft)))

	app.InitState(DebugStateOff)

	app.AddSystems(byke.OnEnter(DebugStateOn), setupDebugCameraSystem)

	app.AddSystems(byke.Update, byke.
		System(renderDebugTextSystem).
		RunIf(byke.InState(DebugStateOn)))

	app.AddSystems(byke.PostUpdate, clearRenderContextMetricsSystem)
}

func setupDebugCameraSystem(commands *byke.Commands) {
	commands.Spawn(
		byke.DespawnOnExitState(DebugStateOn),
		Camera{Order: math.MaxInt},
		RenderLayersOf(31),
		ClearColor{Color: ColorSRGBA(0, 0, 0, 0)},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0, 0},
			ScalingMode:    ScalingModeFixedVertical{ViewportHeight: 600},
		},
	)

	commands.Spawn(
		byke.DespawnOnExitState(DebugStateOn),
		debugMetricsTextMaker{},
		RenderLayersOf(31),
		AnchorTopLeft,
		TransformFromXY(16, 600),
		Text{
			Text:  "Foo\nbar\nbla",
			Color: ColorSRGB(1.0, 0.0, 0.5),
			Size:  12.0,
		},
	)
}

func printCameraGlobalPositionSystem(
	query byke.Query[struct {
		_         byke.With[Camera]
		Transform GlobalTransform
	}],
) {
	for item := range query.Items() {
		slog.Info(
			"Camera",
			slog.Any("position", item.Transform.Affine.Translation()),
		)
	}
}

func toggleDebugStateSystem(
	state byke.State[DebugState],
	nextState *byke.NextState[DebugState],
) {
	if state.Current() == DebugStateOn {
		nextState.Set(DebugStateOff)
	} else {
		nextState.Set(DebugStateOn)
	}
}

func renderDebugTextSystem(
	ctx *RenderContext,
	query byke.Query[struct {
		_    byke.With[debugMetricsTextMaker]
		Text *Text
	}],
) {
	for item := range query.Items() {
		item.Text.Text = ctx.Metrics.String()
	}
}
