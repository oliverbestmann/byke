package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed mesh3d-standard-material.wgsl
var standardMaterialShaderCode string

var standardMaterialShaderCache = map[shaderKey]*ShaderDef{}

type shaderKey struct {
	Texture   bool
	NormalMap bool
}

type StandardMaterial struct {
	byke.Component[StandardMaterial]

	// Tint tints the mesh color rendering
	Tint Color

	// Texture is an optional texture to apply to the mesh. This requires the
	// VertexAttributeUV to be set. Will be ignored if UVs are not set
	Texture *Texture

	// NormalMap is an optional normal map texture
	NormalMap *Texture
}

func (m StandardMaterial) Shader() *ShaderDef {
	key := shaderKey{
		Texture:   m.Texture != nil,
		NormalMap: m.NormalMap != nil,
	}

	cached, ok := standardMaterialShaderCache[key]
	if ok {
		return cached
	}

	var values = ShaderValues{}
	values.Define("MESH3D_COLOR_HAS_NORMALMAP", key.NormalMap)
	values.Define("MESH3D_COLOR_HAS_TEXTURE", key.Texture)

	shader := &ShaderDef{
		Label:         "standard material shader",
		Source:        standardMaterialShaderCode,
		VertexEntry:   "vs_main",
		FragmentEntry: "fs_main",
		Values:        values,
	}

	standardMaterialShaderCache[key] = shader
	return shader
}

func (m StandardMaterial) BindingsLayout() []wgpu.BindGroupLayoutEntry {
	if m.Texture == nil {
		return nil
	}

	return []wgpu.BindGroupLayoutEntry{
		BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
		BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
	}
}

func (m StandardMaterial) Bindings() []wgpu.BindGroupEntry {
	if m.Texture == nil {
		return nil
	}

	return Sequential(
		BindingTextureView(m.Texture.TextureView),
		BindingSampler(m.Texture.Sampler),
	)
}

func (m StandardMaterial) WriteUniforms(w *wgsl.StructWriter) {
	w.AppendVec4f(m.Tint.ToVec())
}
