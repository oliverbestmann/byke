package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed mesh-standard-material.wgsl
var standardMaterialShaderCode string

// DebugNormals can be set to true to output normals instead of colors
var DebugNormals bool

type StandardMaterial struct {
	byke.Component[StandardMaterial]

	// common material values
	MaterialValues

	// Texture is an optional texture to apply to the mesh. This requires the
	// VertexAttributeUV to be set. Will be ignored if UVs are not set
	Texture *Texture

	// The emissive texture if any
	EmissiveTexture *Texture

	// NormalTexture is an optional normal map texture
	NormalTexture *Texture

	// The occlusion texture. Only Red channel is used
	OcclusionTexture *Texture

	// Texture for roughness (green) & metallic (blue)
	RoughnessMetallicTexture *Texture

	// BaseColor tints the mesh color rendering
	BaseColor Color

	// Optional emissive scale. This will be applied to the texture and added to the result,
	// unaffected by lighting. If the material has an EmissiveTexture, it will multiply
	// by the EmissiveTexture value
	EmissiveScale glm.Vec3f

	// Metallic value, zero to one.
	Metallic float32

	// PerceptualRoughness, zero to one.
	// You should probably set this to a non-zero value.
	// A good default is 1.0
	PerceptualRoughness float32
}

func (m StandardMaterial) Shader() *ShaderDef {
	values := ShaderValues{}
	values.Define("MESH3D_MAT_HAS_TEXTURE", m.Texture != nil)
	values.Define("MESH3D_MAT_HAS_NORMAL", m.NormalTexture != nil)
	values.Define("MESH3D_MAT_HAS_EMISSIVE", m.EmissiveTexture != nil)
	values.Define("MESH3D_MAT_HAS_OCCLUSION", m.OcclusionTexture != nil)
	values.Define("MESH3D_MAT_HAS_RMTEX", m.RoughnessMetallicTexture != nil)

	values.Define("ALPHAMODE_OPAQUE", m.AlphaMode == AlphaModeOpaque)
	values.Define("ALPHAMODE_MASK", m.AlphaMode == AlphaModeMask)
	values.Define("ALPHAMODE_ALPHA_TO_COVERAGE", m.AlphaMode == AlphaModeAlphaToCoverage)
	values.Define("ALPHAMODE_BLEND", m.AlphaMode == AlphaModeBlend)

	values.Define("LIGHTING", true)

	values.Define("DEBUG_NORMALS", DebugNormals)

	return &ShaderDef{
		Label:         "standard material shader",
		Source:        standardMaterialShaderCode,
		VertexEntry:   "vs_main",
		FragmentEntry: "fs_main",
		Values:        values,
	}
}

func (m StandardMaterial) Specialize(pipeline *RenderPipelineDescriptor) {
	m.MaterialValues.Specialize(pipeline)

	var bindings []wgpu.BindGroupLayoutEntry
	bindings = append(bindings, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false))
	bindings = append(bindings, m.BindingsLayout()...)

	pipeline.Layout = append(pipeline.Layout, SequentialLayoutWithLabel("StandardMaterial", bindings...))
}

func (m StandardMaterial) BindingsLayout() []wgpu.BindGroupLayoutEntry {
	var entries []wgpu.BindGroupLayoutEntry

	if m.Texture != nil {
		entries = append(
			entries,
			Indexed(1, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(2, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	if m.NormalTexture != nil {
		entries = append(
			entries,
			Indexed(3, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(4, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	if m.EmissiveTexture != nil {
		entries = append(
			entries,
			Indexed(5, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(6, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	if m.OcclusionTexture != nil {
		entries = append(
			entries,
			Indexed(7, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(8, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	if m.RoughnessMetallicTexture != nil {
		entries = append(
			entries,
			Indexed(9, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(10, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	return entries
}

func (m StandardMaterial) Bindings() []wgpu.BindGroupEntry {
	var entries []wgpu.BindGroupEntry

	if m.Texture != nil {
		entries = append(
			entries,
			Indexed(1, BindingTextureView(m.Texture.TextureView)),
			Indexed(2, BindingSampler(m.Texture.Sampler)),
		)
	}

	if m.NormalTexture != nil {
		entries = append(
			entries,
			Indexed(3, BindingTextureView(m.NormalTexture.TextureView)),
			Indexed(4, BindingSampler(m.NormalTexture.Sampler)),
		)
	}

	if m.EmissiveTexture != nil {
		entries = append(
			entries,
			Indexed(5, BindingTextureView(m.EmissiveTexture.TextureView)),
			Indexed(6, BindingSampler(m.EmissiveTexture.Sampler)),
		)
	}

	if m.OcclusionTexture != nil {
		entries = append(
			entries,
			Indexed(7, BindingTextureView(m.OcclusionTexture.TextureView)),
			Indexed(8, BindingSampler(m.OcclusionTexture.Sampler)),
		)
	}

	if m.RoughnessMetallicTexture != nil {
		entries = append(
			entries,
			Indexed(9, BindingTextureView(m.RoughnessMetallicTexture.TextureView)),
			Indexed(10, BindingSampler(m.RoughnessMetallicTexture.Sampler)),
		)
	}

	return entries
}

func (m StandardMaterial) WriteUniforms(w *wgsl.StructWriter) {
	w.AppendVec4f(m.BaseColor.ToVec())
	w.AppendVec3f(m.EmissiveScale)
	w.AppendUint(uint32(boolToInt(m.DoubleSided)))
	w.AppendFloat32(m.AlphaCutoff)
	w.AppendFloat32(m.Metallic)
	w.AppendFloat32(m.PerceptualRoughness)
}

func (m StandardMaterial) BindGroupKey() MaterialBindGroupKey {
	var hash Hash = 0xC2ACE5D3D65CE2C6
	hash.Pointer(m.Texture)
	hash.Pointer(m.EmissiveTexture)
	hash.Pointer(m.NormalTexture)
	hash.Pointer(m.OcclusionTexture)
	hash.Pointer(m.RoughnessMetallicTexture)
	hash.Int(m.MaterialValues.BindGroupKey())
	return MaterialBindGroupKey(hash)
}

func (m StandardMaterial) PipelineKey() MaterialPipelineKey {
	var key uint64

	key |= boolToUint64(m.Texture != nil) << 0
	key |= boolToUint64(m.EmissiveTexture != nil) << 1
	key |= boolToUint64(m.NormalTexture != nil) << 2
	key |= boolToUint64(m.OcclusionTexture != nil) << 3
	key |= boolToUint64(m.RoughnessMetallicTexture != nil) << 4

	var hash Hash = 0xC2ACE5D3D65CE2C6
	hash.Int(key)
	hash.Int(m.MaterialValues.PipelineKey())
	return MaterialPipelineKey(hash)
}

func boolToUint64(value bool) uint64 {
	return uint64(boolToInt(value))
}
