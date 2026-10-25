package shared

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

// hintLayer keeps the hint out of the cameras of the example
const hintLayer = 31

// PluginHint shows a short note about the controls in the bottom left corner,
// drawn by its own camera on top of the example.
//
// It does nothing in tests: the frame dump only captures one camera, so a second
// camera would make snapshots depend on which camera is dumped last.
func PluginHint(text string) byke.Plugin {
	return func(app *byke.App) {
		if InTesting {
			return
		}

		app.AddSystems(byke.Startup, func(commands *byke.Commands) {
			commands.Spawn(
				byke2d.Camera{Order: 1},
				byke2d.Camera2d,
				byke2d.ClearColor{Color: byke2d.ColorSRGBA(0, 0, 0, 0)},
				byke2d.OrthographicProjection{ViewportOrigin: glm.Vec2f{0, 0}},
				byke2d.RenderLayersOf(hintLayer),
			)

			commands.Spawn(
				byke2d.AnchorBottomLeft,
				byke2d.TransformFromXYZ(16, 16, 0),
				byke2d.Text{Text: text, Size: 16, Color: byke2d.ColorSRGBA(1, 1, 1, 0.8)},
				byke2d.RenderLayersOf(hintLayer),
			)
		})
	}
}

// HintFirstPerson describes the controls of byke2d.FirstPersonViewController.
const HintFirstPerson = "Click to look around, WASD and Q/E to move"
