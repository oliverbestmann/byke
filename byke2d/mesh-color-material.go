package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed mesh-color-material.wgsl
var colorMaterialShaderCode string

var _ Material = ColorMaterial{}
var _ ResolvableMaterial = &ColorMaterial{}

type ColorMaterial struct {
	byke.ComparableComponent[ColorMaterial]

	ColorMaterialBindGroup

	// Color tints the mesh color rendering
	Color Color

	AlphaCutoff float32

	resolved *MaterialBindGroupHandle
}

func (m *ColorMaterial) ResolveBindGroup(cache *MaterialBindGroupHandleCache) {
	cache.ResolveBindGroup(&m.resolved, m.ColorMaterialBindGroup)
}

type ColorMaterialBindGroup struct {
	MaterialValues

	// Texture is an optional texture to apply to the mesh. This requires the
	// VertexAttributeUV to be set. Will be ignored if UVs are not set
	Texture *Texture
}

func (m *ColorMaterialBindGroup) Shader() *ShaderDef {
	values := ShaderValues{}
	values.Set("MESH3D_MAT_HAS_TEXTURE", m.Texture != nil)

	values.Set("ALPHAMODE_OPAQUE", m.AlphaMode == AlphaModeOpaque)
	values.Set("ALPHAMODE_MASK", m.AlphaMode == AlphaModeMask)
	values.Set("ALPHAMODE_ALPHA_TO_COVERAGE", m.AlphaMode == AlphaModeAlphaToCoverage)

	values.Set("LIGHTING", false)

	return &ShaderDef{
		Label:         "standard material shader",
		Source:        colorMaterialShaderCode,
		VertexEntry:   "vs_main",
		FragmentEntry: "fs_main",
		Values:        values,
	}
}

func (m *ColorMaterialBindGroup) Specialize(pipeline *RenderPipelineDescriptor) {
	m.MaterialValues.Specialize(pipeline)

	var bindings []wgpu.BindGroupLayoutEntry
	bindings = append(bindings, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false))
	bindings = append(bindings, m.BindingsLayout()...)

	pipeline.Layout = append(pipeline.Layout, SequentialLayoutWithLabel("StandardMaterial", bindings...))
}

func (m *ColorMaterialBindGroup) BindingsLayout() []wgpu.BindGroupLayoutEntry {
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

func (m *ColorMaterialBindGroup) Bindings() []wgpu.BindGroupEntry {
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
	w.AppendVec4f(m.Color.ToVec())
	w.AppendFloat32(m.AlphaCutoff)
}

func (m ColorMaterial) BindGroup() *MaterialBindGroupHandle {
	return m.resolved
}

func (m *ColorMaterialBindGroup) PipelineKey() MaterialPipelineKey {
	var hash Hash = 0xC2ACE5D3D65CE2C6
	hash.Bool(m.Texture != nil)
	hash.Int(m.MaterialValues.BindGroupKey())
	return MaterialPipelineKey(hash)
}
