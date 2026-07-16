package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed mesh3d-standard-material.wgsl
var standardMaterialShaderCode string

var standardMaterialShaderCache = map[shaderKey]*ShaderDef{}

type shaderKey struct {
	Texture   bool
	Normal    bool
	Emissive  bool
	Occlusion bool
	AlphaMode AlphaMode
}

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

	// FrontFace defaults to wgpu.FrontFaceCCW
	FrontFace wgpu.FrontFace

	// AlphaMode decides on the way this material handles alpha values.
	AlphaMode AlphaMode

	// AlphaCutoff is used with AlphaModeMask to define the
	// cutoff for the alpha value.
	AlphaCutoff float32

	// DoubleSided enables double-sided lighting.
	// Need to flip the backface vertex in pixel shader
	DoubleSided bool
}

func (m StandardMaterial) Shader() *ShaderDef {
	key := shaderKey{
		Texture:   m.Texture != nil,
		Normal:    m.NormalTexture != nil,
		Emissive:  m.EmissiveTexture != nil,
		Occlusion: m.OcclusionTexture != nil,
		AlphaMode: m.AlphaMode,
	}

	cached, ok := standardMaterialShaderCache[key]
	if ok {
		return cached
	}

	values := ShaderValues{}
	values.Define("MESH3D_MAT_HAS_NORMAL", key.Normal)
	values.Define("MESH3D_MAT_HAS_TEXTURE", key.Texture)
	values.Define("MESH3D_MAT_HAS_EMISSIVE", key.Emissive)
	values.Define("MESH3D_MAT_HAS_OCCLUSION", key.Occlusion)

	values.Define("ALPHAMODE_OPAQUE", key.AlphaMode == AlphaModeOpaque)
	values.Define("ALPHAMODE_MASK", key.AlphaMode == AlphaModeMask)
	values.Define("ALPHAMODE_ALPHA_TO_COVERAGE", key.AlphaMode == AlphaModeAlphaToCoverage)

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
	var hash = standardMaterialHashSeed
	hash.Pointer(m.Texture)
	hash.Pointer(m.EmissiveTexture)
	hash.Pointer(m.NormalTexture)
	hash.Pointer(m.OcclusionTexture)
	hash.Int(m.FrontFace)
	hash.Int(m.AlphaMode)
	hash.Bool(m.DoubleSided)
	return MaterialBindGroupKey(hash)
}

func (m StandardMaterial) IsSameBindGroup(other Material) bool {
	matOther, ok := other.(StandardMaterial)
	if !ok {
		return false
	}

	return m.BindGroupKey() == matOther.BindGroupKey()
}

func (m StandardMaterial) Specialize(pipeline *RenderPipelineDescriptor) {
	pipeline.Primitive.FrontFace = frontFaceOf(m.FrontFace)

	if m.DoubleSided {
		// disable culling so we can render both sides of the triangles
		pipeline.Primitive.CullMode = wgpu.CullModeNone
	}

	if m.AlphaMode == AlphaModeAlphaToCoverage {
		// we could have alpha values in the image that we want to have
		// masked out
		pipeline.Multisample.AlphaToCoverageEnabled = true
	}
}

const standardMaterialHashSeed Hash = 0xC2ACE5D3D65CE2C6
