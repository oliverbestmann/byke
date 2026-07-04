package byke2d

import (
	_ "embed"
	"unsafe"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed mesh2d-color-material.wgsl
var colorMaterialShaderCode string

var colorMaterialShader = &ShaderDef{
	Label:         "color material shader",
	Source:        colorMaterialShaderCode,
	VertexEntry:   "vs_main",
	FragmentEntry: "fs_main",
}

var colorMaterialShaderWithTexture = &ShaderDef{
	Label:         "color material shader",
	Source:        colorMaterialShaderCode,
	VertexEntry:   "vs_main",
	FragmentEntry: "fs_main",
	Values: ShaderValues{
		"MESH2D_COLOR_HAS_TEXTURE": "true",
	},
}

type ColorMaterial struct {
	byke.Component[ColorMaterial]

	// Tint tints the mesh color rendering
	Tint Color

	// Texture is an optional texture to apply to the mesh. This requires the
	// VertexAttributeUV to be set. Will be ignored if UVs are not set
	Texture *Texture
}

func (m ColorMaterial) Shader() *ShaderDef {
	if m.Texture != nil {
		return colorMaterialShaderWithTexture
	}

	return colorMaterialShader
}

func (m ColorMaterial) BindingsLayout() []wgpu.BindGroupLayoutEntry {
	if m.Texture == nil {
		return nil
	}

	return []wgpu.BindGroupLayoutEntry{
		BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
		BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
	}
}

func (m ColorMaterial) Bindings() []wgpu.BindGroupEntry {
	if m.Texture == nil {
		return nil
	}

	return Sequential(
		BindingTextureView(m.Texture.TextureView),
		BindingSampler(m.Texture.Sampler),
	)
}

func (m ColorMaterial) WriteUniforms(w *wgsl.StructWriter) {
	w.AppendVec4f(m.Tint.ToVec())
}

func (m ColorMaterial) BindGroupKey() MaterialBindGroupKey {
	return colorMaterialKey{Texture: m.Texture}
}

type colorMaterialKey struct {
	Texture *Texture
}

func (c colorMaterialKey) SortValue() uint64 {
	return uint64(uintptr(unsafe.Pointer(c.Texture)))
}
