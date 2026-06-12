package byke2d

import (
	"slices"
	"strconv"
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh3dPipelineConfig struct {
	Shader           *ShaderDef
	Format           wgpu.TextureFormat
	Attributes       []VertexAttribute
	MaterialBindings []wgpu.BindGroupLayoutEntry
	SampleCount      uint32
	Skinned          bool
}

func (m mesh3dPipelineConfig) EqualTo(other PipelineConfig) bool {
	otherConfig, ok := other.(mesh3dPipelineConfig)
	return ok &&
		m.Shader.EqualTo(otherConfig.Shader) &&
		m.Format == otherConfig.Format &&
		m.SampleCount == otherConfig.SampleCount &&
		m.Skinned == otherConfig.Skinned &&
		slices.Equal(m.Attributes, otherConfig.Attributes) &&
		slices.Equal(m.MaterialBindings, otherConfig.MaterialBindings)
}

func (m mesh3dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	values := m.Shader.Values.Clone()

	var instanceAttrs vertexAttributeOffsets

	buffers := []wgpu.VertexBufferLayout{
		{
			// per instance: model to world transform
			ArrayStride: 48,
			StepMode:    wgpu.VertexStepModeInstance,
			Attributes: []wgpu.VertexAttribute{
				// affine [4]vec3f
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
			},
		},
		{
			// per vertex: x, y, z
			ArrayStride: 12,
			StepMode:    wgpu.VertexStepModeVertex,
			Attributes: []wgpu.VertexAttribute{
				{
					Format:         wgpu.VertexFormatFloat32x3,
					ShaderLocation: 4,
					Offset:         0,
				},
			},
		},
	}

	var attrShaderLocation uint32 = 5
	for _, attr := range m.Attributes {
		buffers = append(buffers, wgpu.VertexBufferLayout{
			ArrayStride: uint64(attr.Format.ByteSize()),
			StepMode:    wgpu.VertexStepModeVertex,
			Attributes: []wgpu.VertexAttribute{
				{
					Format:         attr.Format,
					ShaderLocation: attrShaderLocation,
				},
			},
		})

		// define the key for the shader to know about it
		key := strings.ToUpper(attr.Name)
		loc := strconv.Itoa(int(attrShaderLocation))
		values.Set("MESH3D_VERTEX_ATTRIBUTES_"+key, loc)

		attrShaderLocation += 1
	}

	values.Define("SKINNED", m.Skinned)

	mod := ctx.Shader(m.Shader.Label, m.Shader.Source, values)

	return RenderPipelineDescriptor{
		Label: "mesh3d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			ViewBindGroupLayout,
			LightsBindGroupLayout,
			SequentialLayout(slices.Clone(m.MaterialBindings)...),
			SkinningBindGroupLayout,
			// no further bindings for now
		},
		Vertex: wgpu.VertexState{
			Module:     mod,
			EntryPoint: m.Shader.VertexEntry,
			Buffers:    buffers,
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			CullMode:  wgpu.CullModeBack,
			FrontFace: wgpu.FrontFaceCW,
		},
		Multisample: multisampleState(m.SampleCount),
		Fragment: &wgpu.FragmentState{
			Module:     mod,
			EntryPoint: m.Shader.FragmentEntry,
			Targets: []wgpu.ColorTargetState{
				{
					Format:    m.Format,
					Blend:     &wgpu.BlendStateReplace,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: wgpu.OptionalBoolTrue,
			DepthCompare:      wgpu.CompareFunctionLessEqual,
		},
	}
}
