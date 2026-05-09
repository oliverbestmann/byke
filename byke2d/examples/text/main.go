package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
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
	app.AddSystems(Update, updateTextSystem)

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
	)

	commands.Spawn(
		TransformFromXY(0, 200),
		AnchorCenter,
		UpdateTextMarker{},
		Text{
			Size:  48.0,
			Color: ColorSRGBA(1, 1, 1, 1.0),
		},
	)

	commands.Spawn(
		TransformFromXY(8, -80),
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
}]) {
	for item := range query.Items() {
		fps := float64(vt.Frames) / vt.Elapsed.Seconds()
		item.Text.Text = fmt.Sprintf("Seconds: %1.2f,\nabout %1.2f fps", vt.Elapsed.Seconds(), fps)
	}
}
