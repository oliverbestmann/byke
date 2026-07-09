package byke2d

import (
	"strconv"
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh3dPipelineConfig struct {
	Format       wgpu.TextureFormat
	VertexLayout VertexLayout
	Material     Material
	SampleCount  uint32
	Skinned      bool
	Morph        bool

	cachedMaterialBindGroupKey uint64
}

func (m mesh3dPipelineConfig) EqualTo(other PipelineConfig) bool {
	otherConfig, ok := other.(mesh3dPipelineConfig)
	return ok &&
		m.Format == otherConfig.Format &&
		m.SampleCount == otherConfig.SampleCount &&
		m.Skinned == otherConfig.Skinned &&
		m.Morph == otherConfig.Morph &&
		m.VertexLayout.EqualTo(otherConfig.VertexLayout) &&
		m.Material.IsSameBindGroup(otherConfig.Material)
}

func (m mesh3dPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	shader := m.Material.Shader()
	values := shader.Values.Clone()

	var bindings []wgpu.BindGroupLayoutEntry
	bindings = append(bindings, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false))
	bindings = append(bindings, m.Material.BindingsLayout()...)

	var instanceAttrs, perVertexAttrs vertexAttributeOffsets

	vblInstances := wgpu.VertexBufferLayout{
		// per instance: model to world transform
		ArrayStride: 60,
		StepMode:    wgpu.VertexStepModeInstance,
		Attributes: []wgpu.VertexAttribute{
			// affine [4]vec3f
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),
			instanceAttrs.Inc(wgpu.VertexFormatFloat32x3),

			// base vertex index
			instanceAttrs.Inc(wgpu.VertexFormatUint32),

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
		vblPerVertex.Attributes = append(
			vblPerVertex.Attributes,
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

	mod := ctx.Shader(shader.Label, shader.Source, values)

	desc := RenderPipelineDescriptor{
		Label: "mesh3d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			MeshViewBindGroupLayout,
			MeshBindGroupLayout,
			SequentialLayout(bindings...),
		},
		Vertex: wgpu.VertexState{
			Module:     mod,
			EntryPoint: shader.VertexEntry,
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
			EntryPoint: shader.FragmentEntry,
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

	m.Material.Specialize(&desc)

	return desc
}

func frontFaceOf(f wgpu.FrontFace) wgpu.FrontFace {
	switch f {
	case wgpu.FrontFaceCW:
		return wgpu.FrontFaceCW

	case wgpu.FrontFaceCCW:
		return wgpu.FrontFaceCCW

	default:
		return wgpu.FrontFaceCCW
	}
}
