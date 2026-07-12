package byke2d

import (
	"slices"
	"strconv"
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh2dPipelineConfig struct {
	Shader           *ShaderDef
	Format           wgpu.TextureFormat
	Attributes       []VertexAttribute
	MaterialBindings []wgpu.BindGroupLayoutEntry
	SampleCount      uint32
}

func (m mesh2dPipelineConfig) EqualTo(other PipelineConfig) bool {
	otherConfig, ok := other.(mesh2dPipelineConfig)
	return ok &&
		m.Shader.EqualTo(otherConfig.Shader) &&
		m.Format == otherConfig.Format &&
		m.SampleCount == otherConfig.SampleCount &&
		slices.Equal(m.Attributes, otherConfig.Attributes) &&
		slices.Equal(m.MaterialBindings, otherConfig.MaterialBindings)
}

func (m mesh2dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	values := m.Shader.Values.Clone()

	var instanceAttrs vertexAttributeOffsets

	buffers := []wgpu.VertexBufferLayout{
		{
			// per instance: model to world transform
			ArrayStride: 52,
			StepMode:    wgpu.VertexStepModeInstance,
			Attributes: []wgpu.VertexAttribute{
				// affine [4]vec3f
				instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),

				// material index
				instanceAttrs.Inc(wgpu.VertexFormatUint32),
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
		values.Set("MESH2D_VERTEX_ATTRIBUTES_"+key, loc)

		attrShaderLocation += 1
	}

	mod := ctx.Shader(m.Shader.Label, m.Shader.Source, values)

	return RenderPipelineDescriptor{
		Label: "mesh2d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			ViewBindGroupLayout,
			SequentialLayout(slices.Clone(m.MaterialBindings)...),
			// no further bindings for now
		},
		Vertex: wgpu.VertexState{
			Module:     mod,
			EntryPoint: m.Shader.VertexEntry,
			Buffers:    buffers,
		},
		Primitive: wgpu.PrimitiveState{
			Topology: wgpu.PrimitiveTopologyTriangleList,
			CullMode: wgpu.CullModeNone,
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
			DepthCompare:      wgpu.CompareFunctionGreater,
		},
	}
}
