package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke"
)

func PluginDebug(app *byke.App) {
	app.AddSystems(byke.Update, printRenderContextMetricsSystem)
	app.AddSystems(byke.PostUpdate, clearRenderContextMetricsSystem)
}

func printRenderContextMetricsSystem(
	ctx *RenderContext,
) {
	fmt.Println(ctx.Metrics.String())
}

func clearRenderContextMetricsSystem(ctx *RenderContext) {
	ctx.Metrics.reset()
}
