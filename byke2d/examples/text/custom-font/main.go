package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// Draws text with a font loaded from the assets folder.
// Text without a Font component uses the built-in sans font.
//
// Font: Press Start 2P, SIL Open Font License, see assets/OFL.txt.

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
		},
	)

	// a font can hold multiple faces, e.g. for fallback glyphs
	font := Font{Faces: ParseFontFaces(assets.Bytes("PressStart2P-Regular.ttf").Await())}

	commands.Spawn(
		TransformFromXY(0, 80),
		font,
		Text{Text: "GAME OVER", Size: 48, Color: ColorSRGB(1, 0.3, 0.3)},
	)

	commands.Spawn(
		TransformFromXY(0, -20),
		font,
		Text{Text: "Press start to continue", Size: 16, Color: ColorWhite},
	)

	commands.Spawn(
		TransformFromXY(0, -100),
		Text{Text: "This text uses the default font", Size: 24, Color: ColorSRGB(0.6, 0.6, 0.6)},
	)
}
