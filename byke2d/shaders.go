package byke2d

import (
	"embed"
	_ "embed"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"time"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/wesl-go"
)

var _ = byke.ValidateComponent[CustomShader]()

//go:embed shaders-lib
var fsShaders embed.FS

func pluginShader(app *byke.App) {
	// read all shader files
	shaders, _ := fs.Sub(fsShaders, "shaders-lib")
	lib, _ := wesl.FilesOf(shaders)

	app.InsertResource(Shaders{
		transpiler: wesl.New(),
		files:      lib,
	})
}

type Shaders struct {
	transpiler *wesl.Transpiler
	files      map[string]string
}

func (s *Shaders) Add(name, source string) {
	ensureMapIsInitialized(&s.files)
	s.files[name] = source
}

func (s *Shaders) Get() wesl.Files {
	return s.files
}

func (s *Shaders) Compile(source string, values ShaderValues) (string, error) {
	files := maps.Clone(s.files)
	maps.Insert(files, maps.All(values.Files))
	files["main.wesl"] = source

	startTime := time.Now()

	wgsl, err := s.transpiler.Transpile("main.wesl", wesl.Options{
		Files:       files,
		Constants:   values.constants,
		Conditions:  values.conditions,
		PackageName: "byke",
	})

	if err != nil {
		return "", fmt.Errorf("transpile source to wgsl: %w", err)
	}

	slog.Debug("Compiled shader source to wgsl", slog.Duration("duration", time.Since(startTime)))

	return wgsl, err
}

type ShaderValues struct {
	// Extra set of files to include
	Files wesl.Files

	conditions map[string]bool
	constants  map[string]any
}

func (v *ShaderValues) Set(name string, enable bool) {
	ensureMapIsInitialized(&v.conditions)
	v.conditions[name] = enable
}

func (v *ShaderValues) EqualTo(other ShaderValues) bool {
	return maps.Equal(v.conditions, other.conditions) &&
		maps.Equal(v.constants, other.constants)
}

func (v *ShaderValues) Clone() ShaderValues {
	return ShaderValues{
		conditions: maps.Clone(v.conditions),
		constants:  maps.Clone(v.constants),
	}
}

type CompileShaderOptions struct {
	Constants  map[string]any
	Conditions map[string]bool
	Files      wesl.Files
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
	Values        ShaderValues
}

func (s *ShaderDef) EqualTo(other *ShaderDef) bool {
	return s == other || (s.Label == other.Label &&
		s.Source == other.Source &&
		s.VertexEntry == other.VertexEntry &&
		s.FragmentEntry == other.FragmentEntry &&
		s.Values.EqualTo(other.Values))
}
