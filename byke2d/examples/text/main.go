package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
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
		defer profile.Start(profile.CPUProfile).Stop()
	}

	slog.SetDefault(slog.New(handler))

	var app App

	// configure assets before loading the plugin
	app.InsertResource(MakeAssetFS(assets))

	app.AddPlugin(PluginRender)
	app.AddSystems(Update, ExitOnEscapeSystem)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, updateTextSystem)

	app.MustRun()
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    &ScalingModeFixedHorizontal{ViewportWidth: 1280},
		},
	)

	commands.Spawn(
		TransformFromXY(0, 200).WithRotationZ(3.14159*-0.1),
		AnchorCenter,
		UpdateTextMarker{},
		Text{
			Size:  48.0,
			Color: ColorSRGBA(1, 1, 1, 1.0),
		},
	)

	commands.Spawn(
		TransformFromXY(8, -80).WithRotationZ(3.14159*0.25),
		AnchorTopLeft,
		Text{
			Size:  24.0,
			Color: ColorSRGBA(1, 0.75, 0.65, 1.0),
			Text:  "This is some\nMultiline text.\nDepending on the font, we also have ligatures\ni.e. compare fi to f i.",
		},
	)

	commands.Spawn(
		TransformFromXY(-8, -80),
		AnchorTopRight,
		Text{
			Size:  24.0,
			Color: ColorSRGBA(1, 0.75, 0.65, 1.0),
			Text:  "Hello すべての world",
		},
	)
}

type UpdateTextMarker struct {
	ImmutableComponent[UpdateTextMarker]
}

func updateTextSystem(vt VirtualTime, query Query[struct {
	_    With[UpdateTextMarker]
	Text *Text
}],
) {
	for item := range query.Items() {
		fps := float64(vt.Frames) / vt.Elapsed.Seconds()
		item.Text.Text = fmt.Sprintf("Seconds: %1.2f,\nabout %1.2f fps", vt.Elapsed.Seconds(), fps)
	}
}
