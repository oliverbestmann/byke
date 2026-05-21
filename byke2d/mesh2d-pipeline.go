package byke2d

import (
	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh2dPipelineConfig struct {
	Format      wgpu.TextureFormat
	SampleCount uint32
	HasColors   bool
}

func (m mesh2dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	var shaderSource = "#import byke2d::mesh2d"

	values := ShaderValues{}
	values.Define("MESH2D_VERTEX_ATTRIBUTES_COLOR", m.HasColors)

	mod := ctx.Shader("mesh2d shader", shaderSource, values)

	var instanceAttrs offsetCalc

	vLayout := []wgpu.VertexBufferLayout{
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
			ArrayStride: 8,
			StepMode:    wgpu.VertexStepModeVertex,
			Attributes: []wgpu.VertexAttribute{
				{
					Format:         wgpu.VertexFormatFloat32x2,
					ShaderLocation: 10,
					Offset:         0,
				},
			},
		},
	}

	if m.HasColors {
		vLayout = append(vLayout, wgpu.VertexBufferLayout{
			ArrayStride: 16,
			StepMode:    wgpu.VertexStepModeVertex,
			Attributes: []wgpu.VertexAttribute{
				{
					Format:         wgpu.VertexFormatFloat32x4,
					ShaderLocation: 11,
					Offset:         0,
				},
			},
		})
	}

	return RenderPipelineDescriptor{
		Label: "mesh2d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			ViewBindGroupLayout,
			// no further bindings for now
		},
		Vertex: wgpu.VertexState{
			Module:     mod,
			EntryPoint: "vs_main",
			Buffers:    vLayout,
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
