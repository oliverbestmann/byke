package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/webgpu/wgpu"
)

func FullscreenShaderVertexState(module *wgpu.ShaderModule) wgpu.VertexState {
	return wgpu.VertexState{
		Module:     module,
		EntryPoint: "fullscreen_vertex_shader",
	}
}
