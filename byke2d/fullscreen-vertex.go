package byke2d

import (
	_ "embed"
)

//go:embed fullscreen-vertex.wgsl
var FullscreenVertexShader string

const FullscreenShaderEntryPoint = "fullscreen_vertex_shader"
