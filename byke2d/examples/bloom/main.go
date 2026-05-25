package main

import (
	"embed"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	"slices"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
	"github.com/pkg/profile"
)

//go:embed assets
var assets embed.FS

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	if runtime.GOOS != "js" {
		defer profile.Start(profile.MemProfileRate(512)).Stop()
	}

	slog.SetDefault(slog.New(handler))

	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(RenderPlugin)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, scaleColorSystem)
	app.AddSystems(Update, System(toggleDebandDitherSystem).RunIf(KeyIsJustPressed(vyn.KeyD)))
	app.AddSystems(Update, System(toggleTonemappingSystem).RunIf(KeyIsJustPressed(vyn.KeyT)))
	app.AddSystems(Update, System(rotateHueSystem).RunIf(KeyIsPressed(vyn.KeyH)))
	app.AddSystems(Update, System(liftSystem).RunIf(KeyIsPressed(vyn.KeyL)))
	app.AddSystems(Update, System(temperatureSystem).RunIf(KeyIsPressed(vyn.KeyU)))

	app.AddSystems(Update, exitSystem)

	app.MustRun()
}

func exitSystem(vt VirtualTime, exit *MessageWriter[AppExit]) {
	if vt.Frames == 1024 {
		exit.Write(AppExitSuccess)
	}
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		NewTransform().WithScaleXY(4, 4),
		Camera{},
		HDR{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeAutoMin{MinWidth: 500, MinHeight: 500},
		},
	)

	asset := assets.Texture("marker.png").Await()
	commands.Spawn(
		Sprite{
			Texture: asset,
			Color:   ColorSRGBA(1, 1, 1, 1),
		},
	)
}

func scaleColorSystem(vt VirtualTime, query Query[struct {
	Sprite *Sprite
}]) {
	for item := range query.Items() {
		c := float32((math.Sin(vt.Elapsed.Seconds())+1.0)*3.0 + 1.0)
		item.Sprite.Color = ColorSRGBA(c, c, c, 1)
	}
}

func toggleDebandDitherSystem(camera Single[*DebandDither]) {
	if *camera.Value == DebandDitherOn {
		*camera.Value = DebandDitherOff
	} else {
		*camera.Value = DebandDitherOn
	}
}

func toggleTonemappingSystem(camera Single[*Tonemapping]) {
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

func rotateHueSystem(vt VirtualTime, camera Single[*ColorGrading]) {
	camera.Value.Global.Hue += vt.DeltaSecs
}

func liftSystem(vt VirtualTime, camera Single[*ColorGrading]) {
	camera.Value.Midtones.Lift += vt.DeltaSecs
}

func temperatureSystem(vt VirtualTime, camera Single[*ColorGrading]) {
	camera.Value.Global.Temperature += vt.DeltaSecs
}
