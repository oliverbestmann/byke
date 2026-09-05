package byke2d

import (
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type meshPipelineConfig struct {
	Format       wgpu.TextureFormat
	VertexLayout VertexLayout
	Material     Material
	SampleCount  uint32
	Skinned      bool
	Morph        bool

	// options to pick the mesh view bind group layout
	MeshView MeshViewBindGroupLayoutOptions
}

func (c meshPipelineConfig) Hash() uint32 {
	h := HashFor[meshPipelineConfig]()
	h.Int(c.Format)
	h.Int(c.VertexLayout.Key())
	h.Int(c.Material.BindGroup().PipelineKey)
	h.Int(c.SampleCount)
	h.Bool(c.Skinned)
	h.Bool(c.Morph)
	return uint32(h)
}

func (c meshPipelineConfig) EqualTo(other PipelineConfig) bool {
	otherConfig, ok := other.(meshPipelineConfig)
	return ok &&
		c.Format == otherConfig.Format &&
		c.SampleCount == otherConfig.SampleCount &&
		c.Skinned == otherConfig.Skinned &&
		c.Morph == otherConfig.Morph &&
		c.VertexLayout.EqualTo(otherConfig.VertexLayout) &&
		c.Material.BindGroup().PipelineKey == otherConfig.Material.BindGroup().PipelineKey &&
		c.MeshView == otherConfig.MeshView
}

func (c meshPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	shader := c.Material.BindGroup().BindGroup.Shader()
	values := shader.Values.Clone()

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
		ArrayStride: uint64(c.VertexLayout.Size()),
		StepMode:    wgpu.VertexStepModeVertex,
	}

	for _, attr := range c.VertexLayout.Attributes {
		vblPerVertex.Attributes = append(
			vblPerVertex.Attributes,
			perVertexAttrs.AtLoc(attr.Location, attr.Format),
		)

		// define the key for the shader to know about it
		key := strings.ToUpper(attr.Name)
		values.Set("MESH3D_VERTEX_ATTRIBUTES_"+key, true)
	}

	buffers := []wgpu.VertexBufferLayout{
		vblInstances,
		vblPerVertex,
	}

	values.Set("SKINNED", c.Skinned)
	values.Set("MORPH", c.Morph)
	values.Set("MESH_ENVMAP_LIGHT", c.MeshView.EnvironmentMapLight)

	mod := ctx.Shader(shader.Label, shader.Source, values)

	desc := RenderPipelineDescriptor{
		Label: "mesh3d pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			MeshViewBindGroupLayout(c.MeshView),
			MeshBindGroupLayout,
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
		Multisample: multisampleState(c.SampleCount),
		Fragment: &wgpu.FragmentState{
			Module:     mod,
			EntryPoint: shader.FragmentEntry,
			Targets: []wgpu.ColorTargetState{
				{
					Format:    c.Format,
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

	c.Material.BindGroup().BindGroup.Specialize(&desc)

	return desc
}
