package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
)

var _ = byke.ValidateComponent[CustomShader]()

func pluginShader(app *byke.App) {
	preCompiler := pre.New()
	registerShaderModules(preCompiler)
	app.InsertResource(preCompiler)
}

//go:embed fullscreen.wgsl
var fullscreenShader string

//go:embed shaders-lib/colors.wgsl
var colorsShader string

//go:embed shaders-lib/math.wgsl
var mathShader string

//go:embed sprite.wgsl
var spritesShader string

func registerShaderModules(preCompiler pre.Compiler) {
	preCompiler.MustAdd(mathShader)
	preCompiler.MustAdd(colorsShader)
	preCompiler.MustAdd(fullscreenShader)
	preCompiler.MustAdd(spritesShader)
}

type CustomShader struct {
	byke.Component[CustomShader]

	// Try re-using the same Instance of ShaderDef
	// for multiple CustomShader components if possible
	Shader *ShaderDef
}

type ShaderDef struct {
	Source        string
	VertexEntry   string
	FragmentEntry string
	Values        pre.Values
}
