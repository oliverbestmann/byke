package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
	shaders_lib "github.com/oliverbestmann/byke/byke2d/shaders-lib"
)

var _ = byke.ValidateComponent[CustomShader]()

func pluginShader(app *byke.App) {
	preCompiler := pre.New()
	registerShaderModules(preCompiler)
	app.InsertResource(preCompiler)
}

func registerShaderModules(preCompiler pre.Compiler) {
	for _, shader := range shaders_lib.All() {
		preCompiler.MustAdd(shader)
	}
}

type CustomShader struct {
	byke.Component[CustomShader]

	// Try re-using the same Instance of ShaderDef
	// for multiple CustomShader components if possible
	Shader *ShaderDef
}

type ShaderDef struct {
	Label         string
	Source        string
	VertexEntry   string
	FragmentEntry string
	Values        pre.Values
}

func (s *ShaderDef) EqualTo(other *ShaderDef) bool {
	return s == other || (s.Label == other.Label &&
		s.Source == other.Source &&
		s.VertexEntry == other.VertexEntry &&
		s.FragmentEntry == other.FragmentEntry &&
		s.Values.EqualTo(other.Values))
}
