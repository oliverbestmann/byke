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
	VertexLayout     VertexLayout
	MaterialBindings []wgpu.BindGroupLayoutEntry
	SampleCount      uint32
	Skinned          bool
	Morph            bool
}

func (m mesh3dPipelineConfig) EqualTo(other PipelineConfig) bool {
	otherConfig, ok := other.(mesh3dPipelineConfig)
	return ok &&
		m.Shader.EqualTo(otherConfig.Shader) &&
		m.Format == otherConfig.Format &&
		m.SampleCount == otherConfig.SampleCount &&
		m.Skinned == otherConfig.Skinned &&
		m.Morph == otherConfig.Morph &&
		m.VertexLayout.EqualTo(otherConfig.VertexLayout) &&
		slices.Equal(m.MaterialBindings, otherConfig.MaterialBindings)
}

func (m mesh3dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	values := m.Shader.Values.Clone()

	var instanceAttrs, perVertexAttrs vertexAttributeOffsets

	vblInstances := wgpu.VertexBufferLayout{
		// per instance: model to world transform
		ArrayStride: 56,
		StepMode:    wgpu.VertexStepModeInstance,
		Attributes: []wgpu.VertexAttribute{
			// affine [4]vec3f
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),

			// material index
			instanceAttrs.Inc(wgpu.VertexFormatUint32),

			// morph info index
			instanceAttrs.Inc(wgpu.VertexFormatUint32),
		},
	}

	vblPerVertex := wgpu.VertexBufferLayout{
		// per vertex: x, y, z
		ArrayStride: uint64(m.VertexLayout.Size()),
		StepMode:    wgpu.VertexStepModeVertex,
	}

	for _, attr := range m.VertexLayout.Attributes {
		vblPerVertex.Attributes = append(vblPerVertex.Attributes,
			perVertexAttrs.AtLoc(attr.Location, attr.Format),
		)

		// define the key for the shader to know about it
		key := strings.ToUpper(attr.Name)
		loc := strconv.Itoa(int(attr.Location))
		values.Set("MESH3D_VERTEX_ATTRIBUTES_"+key, loc)
	}

	buffers := []wgpu.VertexBufferLayout{
		vblInstances,
		vblPerVertex,
	}

	values.Define("SKINNED", m.Skinned)
	values.Define("MORPH", m.Morph)

	mod := ctx.Shader(m.Shader.Label, m.Shader.Source, values)

	return RenderPipelineDescriptor{
		Label: "mesh3d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			MeshViewBindGroupLayout,
			MeshBindGroupLayout,
			SequentialLayout(slices.Clone(m.MaterialBindings)...),
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
