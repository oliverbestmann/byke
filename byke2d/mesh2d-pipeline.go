package byke2d

import (
	"strconv"
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh2dPipelineConfig struct {
	Shader           *ShaderDef
	Format           wgpu.TextureFormat
	Attributes       ArraySlice[VertexAttribute]
	MaterialBindings ArraySlice[wgpu.BindGroupLayoutEntry]
	SampleCount      uint32
}

func (m mesh2dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
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
	for _, attr := range m.Attributes.AsSlice() {
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
			SequentialLayout(m.MaterialBindings.AsSlice()...),
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
	}
}
