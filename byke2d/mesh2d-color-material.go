package byke2d

import (
	_ "embed"

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

	// FrontFace defaults to wgpu.FrontFaceCCW
	FrontFace wgpu.FrontFace

	// AlphaMode decides on the way this material handles alpha values.
	AlphaMode AlphaMode

	// AlphaCutoff is used with AlphaModeMask to define the
	// cutoff for the alpha value.
	AlphaCutoff float32
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
	w.AppendFloat32(m.AlphaCutoff)
}

func (m ColorMaterial) BindGroupKey() MaterialBindGroupKey {
	var hash = colorMaterialHashSeed
	hash.Pointer(m.Texture)
	hash.Int(m.FrontFace)
	hash.Int(m.AlphaMode)
	return MaterialBindGroupKey(hash)
}

func (m ColorMaterial) IsSameBindGroup(other Material) bool {
	matOther, ok := other.(ColorMaterial)
	if !ok {
		return false
	}

	return m.BindGroupKey() == matOther.BindGroupKey()
}

func (m ColorMaterial) Specialize(pipeline *RenderPipelineDescriptor) {
	pipeline.Primitive.FrontFace = frontFaceOf(m.FrontFace)

	if m.AlphaMode == AlphaModeAlphaToCoverage {
		// we could have alpha values in the image that we want to have
		// masked out
		pipeline.Multisample.AlphaToCoverageEnabled = true
	}

	if m.AlphaMode == AlphaModeBlend {
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStateAlphaBlending
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse
	}

	if m.AlphaMode == AlphaModeAdd {
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStateAdd
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse
	}
}

const colorMaterialHashSeed Hash = 0x5C36C6CE2CA4FD4F
