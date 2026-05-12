package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/webgpu/wgpu"
)

var FullscreenShaderVertexState = wgpu.VertexState{
	// will be inserted by the the Pipelines[T] cache
	Module:     nil,
	EntryPoint: "fullscreen_vertex_shader",
}
