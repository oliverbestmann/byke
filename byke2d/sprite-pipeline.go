package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

type renderSpritePipelineConfig struct {
	Shader      *ShaderDef
	Format      wgpu.TextureFormat
	SampleCount uint32
}

func (r renderSpritePipelineConfig) EqualTo(other PipelineConfig) bool {
	return r == other
}

func (r renderSpritePipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	shaderLabel := "Sprite"
	shaderSource := "#import byke2d::sprite"
	entryVertex := "vs_main"
	entryFragment := "fs_main"
	var shaderValues ShaderValues

	if r.Shader != nil {
		shaderLabel = valueOr(r.Shader.Label, "Custom Sprite Shader")
		shaderSource = r.Shader.Source
		shaderValues = r.Shader.Values
		entryVertex = valueOr(r.Shader.VertexEntry, entryVertex)
		entryFragment = valueOr(r.Shader.FragmentEntry, entryFragment)
	}

	module := ctx.Shader(shaderLabel, shaderSource, shaderValues)

	var offset vertexAttributeOffsets

	return RenderPipelineDescriptor{
		Label: "sprite",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			ViewBindGroupLayout,
			layoutSpriteTextures,
		},
		Vertex: wgpu.VertexState{
			Module:     module,
			EntryPoint: entryVertex,
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 84,
					StepMode:    wgpu.VertexStepModeInstance,
					Attributes: []wgpu.VertexAttribute{
						offset.Inc(wgpu.VertexFormatFloat32x3),
						offset.Inc(wgpu.VertexFormatFloat32x3),
						offset.Inc(wgpu.VertexFormatFloat32x3),
						offset.Inc(wgpu.VertexFormatFloat32x3),
						offset.Inc(wgpu.VertexFormatFloat32x2),
						offset.Inc(wgpu.VertexFormatFloat32x2),
						offset.Inc(wgpu.VertexFormatFloat32x4),
						offset.Inc(wgpu.VertexFormatUint32),
					},
				},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     module,
			EntryPoint: entryFragment,
			Targets: []wgpu.ColorTargetState{
				{
					Format:    r.Format,
					Blend:     &wgpu.BlendStateAlphaBlending,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},

		Multisample: multisampleState(r.SampleCount),

		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: wgpu.OptionalBoolFalse,
			DepthCompare:      wgpu.CompareFunctionLess,
		},
	}
}
