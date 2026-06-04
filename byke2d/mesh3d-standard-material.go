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
	NormalMap bool
	Emissive  bool
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

	// NormalMap is an optional normal map texture
	NormalMap *Texture
}

func (m StandardMaterial) Shader() *ShaderDef {
	key := shaderKey{
		Texture:   m.Texture != nil,
		NormalMap: m.NormalMap != nil,
		Emissive:  m.EmissiveTexture != nil,
	}

	cached, ok := standardMaterialShaderCache[key]
	if ok {
		return cached
	}

	var values = ShaderValues{}
	values.Define("MESH3D_COLOR_HAS_NORMALMAP", key.NormalMap)
	values.Define("MESH3D_COLOR_HAS_TEXTURE", key.Texture)
	values.Define("MESH3D_COLOR_HAS_EMISSIVE", key.Texture)

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
		entries = append(entries,
			Indexed(1, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(2, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	if m.EmissiveTexture != nil {
		entries = append(entries,
			Indexed(5, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(6, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	return entries
}

func (m StandardMaterial) Bindings() []wgpu.BindGroupEntry {
	var entries []wgpu.BindGroupEntry

	if m.Texture != nil {
		entries = append(entries,
			BindingTextureView(m.Texture.TextureView),
			BindingSampler(m.Texture.Sampler),
		)
	}

	if m.EmissiveTexture != nil {
		entries = append(entries,
			BindingTextureView(m.EmissiveTexture.TextureView),
			BindingSampler(m.EmissiveTexture.Sampler),
		)
	}

	return entries
}

func (m StandardMaterial) WriteUniforms(w *wgsl.StructWriter) {
	w.AppendVec4f(m.Tint.ToVec())
	w.AppendVec3f(m.EmissiveScale)
}
