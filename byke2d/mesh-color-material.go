package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed mesh-color-material.wgsl
var colorMaterialShaderCode string

type ColorMaterial struct {
	byke.Component[ColorMaterial]

	// Tint tints the mesh color rendering
	Tint Color

	// Texture is an optional texture to apply to the mesh. This requires the
	// VertexAttributeUV to be set. Will be ignored if UVs are not set
	Texture *Texture

	MaterialValues
}

func (m ColorMaterial) Shader() *ShaderDef {
	values := ShaderValues{}
	values.Define("MESH3D_MAT_HAS_TEXTURE", m.Texture != nil)

	values.Define("ALPHAMODE_OPAQUE", m.AlphaMode == AlphaModeOpaque)
	values.Define("ALPHAMODE_MASK", m.AlphaMode == AlphaModeMask)
	values.Define("ALPHAMODE_ALPHA_TO_COVERAGE", m.AlphaMode == AlphaModeAlphaToCoverage)

	values.Define("LIGHTING", false)

	return &ShaderDef{
		Label:         "standard material shader",
		Source:        colorMaterialShaderCode,
		VertexEntry:   "vs_main",
		FragmentEntry: "fs_main",
		Values:        values,
	}
}

func (m ColorMaterial) BindingsLayout() []wgpu.BindGroupLayoutEntry {
	var entries []wgpu.BindGroupLayoutEntry

	if m.Texture != nil {
		entries = append(
			entries,
			Indexed(1, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(2, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	return entries
}

func (m ColorMaterial) Bindings() []wgpu.BindGroupEntry {
	var entries []wgpu.BindGroupEntry

	if m.Texture != nil {
		entries = append(
			entries,
			Indexed(1, BindingTextureView(m.Texture.TextureView)),
			Indexed(2, BindingSampler(m.Texture.Sampler)),
		)
	}

	return entries
}

func (m ColorMaterial) WriteUniforms(w *wgsl.StructWriter) {
	w.AppendVec4f(m.Tint.ToVec())
	w.AppendFloat32(m.AlphaCutoff)
}

func (m ColorMaterial) BindGroupKey() MaterialBindGroupKey {
	var hash Hash = 0xEA55D3ABE75DF54F
	hash.Pointer(m.Texture)
	hash.Int(m.MaterialValues.BindGroupKey())
	return MaterialBindGroupKey(hash)
}

func (m ColorMaterial) PipelineKey() MaterialPipelineKey {
	var hash Hash = 0xC2ACE5D3D65CE2C6
	hash.Bool(m.Texture != nil)
	hash.Int(m.MaterialValues.BindGroupKey())
	return MaterialPipelineKey(hash)
}
