package main

import (
	"embed"
	"fmt"
	"log/slog"
	"math"
	"os"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/glm"
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
	app.AddSystems(Update, updateTextSystem, updateTextTransformSystem)
	// app.AddSystems(Update, updateCamera)

	app.MustRun()
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		TransformFromXY(200, 200).WithScaleXY(1, 1),
		Camera{},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			// ScalingMode:    ScalingModeFixedHorizontal{ViewportWidth: 1000.0},
			ScalingMode: ScalingModeWindowSize{},
			Scale:       1.0,
		},
	)

	asset := assets.Texture("coordinates.png").Await()
	commands.Spawn(
		Sprite{Texture: asset, Color: wx.ColorSRGBA(1, 1, 1, 0.25)},
	)

	commands.Spawn(
		TransformFromXY(200, 200),
		Sprite{Texture: asset, CustomSize: Some(glm.Vec2f{64, 64})},
		AnchorCenter,
	)

	commands.Spawn(
		TransformFromXY(200, 200),
		AnchorCenter,
		Text{
			Text:  "Hello World!",
			Size:  48.0,
			Color: wx.ColorSRGBA(1, 1, 1, 1.0),
		},
	)
}

func updateTextSystem(vt VirtualTime, query Query[*Text]) {
	for item := range query.Items() {
		fps := float64(vt.Frames) / vt.Elapsed.Seconds()
		item.Text = fmt.Sprintf("Seconds: %1.2f, about %1.2f fps. fi f i", vt.Elapsed.Seconds(), fps)
	}
}

func updateCamera(vt VirtualTime, query Query[struct {
	Projection *OrthographicProjection
}]) {
	for item := range query.Items() {
		size := (math.Sin(vt.Elapsed.Seconds())+1)*0.5 + 1
		item.Projection.Scale = float32(size)
	}
}

func updateTextTransformSystem(mouse MouseCursor,
	cameraQuery Single[struct {
		_          With[Camera]
		Projection OrthographicProjection
		Transform  GlobalTransform
		ViewTarget *ViewTarget
	}],

	query Query[struct {
		_         With[Text]
		Transform *Transform
	}],
) {
	/*
		viewportSize := cameraQuery.Value.Projection.ScalingMode.ViewportSize(cameraQuery.Value.ViewTarget.Size.XY())
		if viewportSize[0] == 0 {
			return
		}

		screenToNDC := glm.Mat3f{}.
			Translate(viewportSize.M.Extend(1).XY()).
			Scale(cameraQuery.Value.Projection.ViewportOrigin.Mul(-1).XY())

		worldToScreen := cameraQuery.Value.Transform.AsMat3f()

		invert := worldToScreen.Mul(screenToNDC).Invert()

		for item := range query.Items() {
			pos := invert.Transform(mouse.Extend(1))
			item.Transform.Translation = pos

			fmt.Println(pos)
		}
	*/
}
