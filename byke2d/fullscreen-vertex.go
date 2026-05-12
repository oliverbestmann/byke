package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed fullscreen-vertex.wgsl
var FullscreenVertexShader string

const FullscreenShaderEntryPoint = "fullscreen_vertex_shader"

var FullscreenShaderPrimitiveState = wgpu.PrimitiveState{
	Topology: wgpu.PrimitiveTopologyTriangleList,
	CullMode: wgpu.CullModeNone,
}
