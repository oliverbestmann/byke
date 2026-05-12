package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke/byke2d/pre"
)

//go:embed fullscreen.wgsl
var fullscreenShader string

//go:embed shaders-lib/colors.wgsl
var colorsShader string

//go:embed shaders-lib/math.wgsl
var mathShader string

func registerShaderModules(preCompiler pre.Compiler) {
	preCompiler.MustAdd(mathShader)
	preCompiler.MustAdd(colorsShader)
	preCompiler.MustAdd(fullscreenShader)
}
