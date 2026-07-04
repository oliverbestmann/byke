package byke2d

import (
	"fmt"
)

func printRenderContextMetricsSystem(
	ctx *RenderContext,
) {
	fmt.Println(ctx.Metrics.String())
}

func clearRenderContextMetricsSystem(ctx *RenderContext) {
	ctx.Metrics.reset()
}
