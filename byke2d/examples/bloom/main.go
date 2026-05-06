package main

import (
	"embed"
	"fmt"
	"log/slog"
	"math"
	"os"
	"slices"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/vyn"
	"github.com/oliverbestmann/pulse/wx"
)

//go:embed assets
var assets embed.FS

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))

	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, scaleColorSystem)
	app.AddSystems(Update, System(toggleDebandDither).RunIf(KeyIsJustPressed(vyn.KeyD)))
	app.AddSystems(Update, System(toggleTonemapping).RunIf(KeyIsJustPressed(vyn.KeyT)))

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		HDR{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			// ScalingMode:    ScalingModeFixed{Viewport: glm.Vec2f{640, 360}},
			ScalingMode: ScalingModeWindowSize{},
			Scale:       4.0,
		},
	)

	asset := assets.Texture("marker.png").Await()
	commands.Spawn(
		Sprite{
			Texture: asset,
			Color:   wx.ColorSRGBA(1, 1, 1, 1),
		},
	)
}

func scaleColorSystem(vt VirtualTime, query Query[struct {
	Sprite *Sprite
}]) {
	for item := range query.Items() {
		c := float32((math.Sin(vt.Elapsed.Seconds())+1.0)*3.0 + 1.0)
		item.Sprite.Color = wx.ColorSRGBA(c, c, c, 1)
	}
}

func toggleDebandDither(camera Single[*DebandDither]) {
	if *camera.Value == DebandDitherOn {
		*camera.Value = DebandDitherOff
	} else {
		*camera.Value = DebandDitherOn
	}
}

func toggleTonemapping(camera Single[*Tonemapping]) {
	mappings := []Tonemapping{
		TonemappingNone,
		TonemappingSomewhatBoringDisplayTransform,
		TonemappingAcesFitted,
		TonemappingReinhard,
		TonemappingReinhardLuminance,
		TonemappingTonyMcMapface,
		TonemappingAgX,
		TonemappingBlenderFilmic,
	}

	idx := slices.Index(mappings, *camera.Value)
	idx = (idx + 1) % len(mappings)
	*camera.Value = mappings[idx]

	fmt.Println("Tonemapping", *camera.Value)
}
