package shared

import (
	"fmt"
	"time"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
)

var _ = byke.ValidateComponent[fpsText]()

type fpsText struct {
	byke.ImmutableComponent[fpsText]
}

// PluginFpsCounter shows the frames per second near the bottom left corner.
// The text is placed at (32, 32, 1), so it needs a camera with its origin at the bottom left.
func PluginFpsCounter(app *byke.App) {
	app.InsertResource(fpsCounter{})
	app.AddSystems(byke.Startup, spawnFpsTextSystem)
	app.AddSystems(byke.Update, updateFpsTextSystem)
}

func spawnFpsTextSystem(commands *byke.Commands) {
	commands.Spawn(
		fpsText{},
		byke2d.AnchorBottomLeft,
		byke2d.TransformFromXYZ(32, 32, 1),
		byke2d.Text{Size: 16.0, Color: byke2d.ColorSRGBA(1, 0, 1, 1)},
	)
}

func updateFpsTextSystem(
	vt byke.VirtualTime,
	counter *fpsCounter,
	text byke.Single[struct {
		_    byke.With[fpsText]
		Text *byke2d.Text
	}],
) {
	fps := counter.Update(vt.Elapsed)
	text.Value.Text.Text = fmt.Sprintf("FPS: %1.2f", fps)
}

type fpsCounter struct {
	// virtual time, so the text is the same in every test run
	timestamps []time.Duration
}

func (c *fpsCounter) Update(now time.Duration) float32 {
	c.timestamps = append(c.timestamps, now)

	if len(c.timestamps) > 120 {
		c.timestamps = c.timestamps[1:]
	}

	return float32(len(c.timestamps)) / float32((now - c.timestamps[0]).Seconds())
}
