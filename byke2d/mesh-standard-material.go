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

type StandardMaterial struct {
	byke.Component[StandardMaterial]

	// Tint tints the mesh color rendering
	Tint Color

	// Texture is an optional texture to apply to the mesh. This requires the
	// VertexAttributeUV to be set. Will be ignored if UVs are not set
	Texture *Texture

	// Optional emissive scale. This will be applied to the texture and added to the result,
	// unaffected by lighting. If the material has an EmissiveTexture, it will multiply
	// by the EmissiveTexture value
	EmissiveScale glm.Vec3f

	// The emissive texture if any
	EmissiveTexture *Texture

	// NormalTexture is an optional normal map texture
	NormalTexture *Texture

	// The occlusion texture if any
	OcclusionTexture *Texture

	// Some common material values
	MaterialValues
}

func (m StandardMaterial) Shader() *ShaderDef {
	values := ShaderValues{}
	values.Define("MESH3D_MAT_HAS_TEXTURE", m.Texture != nil)
	values.Define("MESH3D_MAT_HAS_NORMAL", m.NormalTexture != nil)
	values.Define("MESH3D_MAT_HAS_EMISSIVE", m.EmissiveTexture != nil)
	values.Define("MESH3D_MAT_HAS_OCCLUSION", m.OcclusionTexture != nil)

	values.Define("ALPHAMODE_OPAQUE", m.AlphaMode == AlphaModeOpaque)
	values.Define("ALPHAMODE_MASK", m.AlphaMode == AlphaModeMask)
	values.Define("ALPHAMODE_ALPHA_TO_COVERAGE", m.AlphaMode == AlphaModeAlphaToCoverage)

	values.Define("LIGHTING", true)

	return &ShaderDef{
		Label:         "standard material shader",
		Source:        standardMaterialShaderCode,
		VertexEntry:   "vs_main",
		FragmentEntry: "fs_main",
		Values:        values,
	}
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

	return entries
}

func (m StandardMaterial) WriteUniforms(w *wgsl.StructWriter) {
	w.AppendVec4f(m.Tint.ToVec())
	w.AppendVec3f(m.EmissiveScale)
	w.AppendUint(uint32(boolToInt(m.DoubleSided)))
	w.AppendFloat32(m.AlphaCutoff)
}

func (m StandardMaterial) BindGroupKey() MaterialBindGroupKey {
	var hash Hash = 0xC2ACE5D3D65CE2C6
	hash.Pointer(m.Texture)
	hash.Pointer(m.EmissiveTexture)
	hash.Pointer(m.NormalTexture)
	hash.Pointer(m.OcclusionTexture)
	hash.Int(m.MaterialValues.BindGroupKey())
	return MaterialBindGroupKey(hash)
}

func (m StandardMaterial) BindGroupLayoutKey() MaterialBindGroupLayoutKey {
	var key uint64

	key |= boolToUint64(m.Texture != nil) << 0
	key |= boolToUint64(m.EmissiveTexture != nil) << 1
	key |= boolToUint64(m.NormalTexture != nil) << 2
	key |= boolToUint64(m.OcclusionTexture != nil) << 3

	var hash Hash = 0xC2ACE5D3D65CE2C6
	hash.Int(key)
	hash.Int(m.MaterialValues.BindGroupKey())
	return MaterialBindGroupLayoutKey(hash)
}

func boolToUint64(value bool) uint64 {
	return uint64(boolToInt(value))
}
