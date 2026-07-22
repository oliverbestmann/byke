package byke2d

import (
	"fmt"
	"log/slog"
	"math"
	"strings"

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
	app.AddSystems(byke.Update, byke.System(dumpTreeSystem).
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
		Camera2d,
		DebugCamera{},
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
		DefaultFontMono(),
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
	alloc *MeshAllocator,
	query byke.Query[struct {
		_    byke.With[debugMetricsTextMaker]
		Text *Text
	}],
) {
	meshAllocatorStats := alloc.Stats()

	for item := range query.Items() {
		var out strings.Builder

		writef := func(format string, args ...any) {
			_, _ = fmt.Fprintf(&out, format, args...)
			out.WriteByte('\n')
		}

		writef("%s", ctx.Metrics.String())
		writef("")
		writef("MeshAllocator")
		writef("  Vertices:        %1.2fkb", float64(meshAllocatorStats.Vertices)/1024)
		writef("  Indices:         %1.2fkb", float64(meshAllocatorStats.Indices)/1024)
		writef("  MorphAttributes: %1.2fkb", float64(meshAllocatorStats.MorphAttributes)/1024)

		item.Text.Text = out.String()
	}
}
