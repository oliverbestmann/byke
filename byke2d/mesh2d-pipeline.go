package byke2d

import (
	"strconv"
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh2dPipelineConfig struct {
	Format      wgpu.TextureFormat
	SampleCount uint32
	HasColors   bool

	Attributes [8]VertexAttribute
}

func (m mesh2dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	var shaderSource = "#import byke2d::mesh2d"

	values := ShaderValues{}

	var instanceAttrs vertexAttributeOffsets

	buffers := []wgpu.VertexBufferLayout{
		{
			ArrayStride: 64,
			StepMode:    wgpu.VertexStepModeInstance,
			Attributes: []wgpu.VertexAttribute{
				// affine [4]vec3f
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),
				instanceAttrs.Inc(12, wgpu.VertexFormatFloat32x3),

				// color
				instanceAttrs.Inc(16, wgpu.VertexFormatFloat32x4),
			},
		},
		{
			ArrayStride: 12,
			StepMode:    wgpu.VertexStepModeVertex,
			Attributes: []wgpu.VertexAttribute{
				{
					Format:         wgpu.VertexFormatFloat32x3,
					ShaderLocation: 5,
					Offset:         0,
				},
			},
		},
	}

	var attrShaderLocation uint32 = 6
	for _, attr := range m.Attributes {
		if attr.Name == "" {
			continue
		}

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

	mod := ctx.Shader("mesh2d shader", shaderSource, values)

	return RenderPipelineDescriptor{
		Label: "mesh2d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			ViewBindGroupLayout,
			// no further bindings for now
		},
		Vertex: wgpu.VertexState{
			Module:     mod,
			EntryPoint: "vs_main",
			Buffers:    buffers,
		},
		Primitive: wgpu.PrimitiveState{
			Topology: wgpu.PrimitiveTopologyTriangleList,
			CullMode: wgpu.CullModeNone,
		},
		Multisample: multisampleState(m.SampleCount),
		Fragment: &wgpu.FragmentState{
			Module:     mod,
			EntryPoint: "fs_main",
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
