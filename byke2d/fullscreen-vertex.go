package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed fullscreen-vertex.wgsl
var fullscreenVertexShader string

func prepareFullscreenShader(def *wgpu.Device) (wgpu.VertexState, wgpu.PrimitiveState) {
	modVertex := def.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "FullscreenShaderVertex",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: fullscreenVertexShader},
	})

	vertexState := wgpu.VertexState{
		Module:     modVertex,
		EntryPoint: "fullscreen_vertex_shader",
	}

	primitveState := wgpu.PrimitiveState{
		Topology: wgpu.PrimitiveTopologyTriangleList,
		CullMode: wgpu.CullModeNone,
	}

	return vertexState, primitveState
}
