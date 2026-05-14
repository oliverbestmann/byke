package shaders_lib

import (
	"embed"
	"io/fs"
)

//go:embed *.wgsl
var shaderFS embed.FS

func All() []string {
	var shaders []string

	paths, _ := fs.Glob(shaderFS, "*.wgsl")
	for _, path := range paths {
		buf, _ := shaderFS.ReadFile(path)
		shaders = append(shaders, string(buf))
	}

	return shaders
}
