package byke2d

import (
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type Material interface {
	Shader() *ShaderDef
	BindingsLayout() []wgpu.BindGroupLayoutEntry
	Bindings() []wgpu.BindGroupEntry
	WriteUniforms(w *wgsl.StructWriter)
}
