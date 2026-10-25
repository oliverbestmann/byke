package byke2d

import (
	_ "embed"
	"strings"

	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed shadow.wgsl
var shadowShader string

type meshPipelineConfig struct {
	Format       wgpu.TextureFormat
	VertexLayout *VertexLayout
	BindGroup    *MaterialBindGroupHandle
	SampleCount  uint32
	Skinned      bool
	Morph        bool
	HasLightmap  bool

	// options to pick the mesh view bind group layout
	MeshView MeshViewBindGroupLayoutOptions
}

func (c meshPipelineConfig) Hash() Hash {
	h := HashFor[meshPipelineConfig]()
	h.Int(c.Format)
	h.Pointer(c.VertexLayout)
	h.Int(c.BindGroup.PipelineKey)
	h.Int(c.SampleCount)
	h.Bool(c.Skinned)
	h.Bool(c.Morph)
	h.Bool(c.HasLightmap)
	h.Int(c.MeshView.Hash())
	return h
}

func (c meshPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	shader := c.BindGroup.BindGroup.Shader()
	values := shader.Values.Clone()

	buffers := meshVertexBuffers(c.VertexLayout, &values)

	values.Set("SKINNED", c.Skinned)
	values.Set("MORPH", c.Morph)
	values.Set("HAS_LIGHTMAP", c.HasLightmap)
	values.Set("MESH_ENVMAP_LIGHT", c.MeshView.EnvironmentMapLight)
	values.Set("MESH_SHADOWS", c.MeshView.Shadows)

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

	c.BindGroup.BindGroup.Specialize(&desc)

	return desc
}

// shadowPipelineConfig renders meshes depth only into a shadow map.
type shadowPipelineConfig struct {
	VertexLayout *VertexLayout
	Skinned      bool
	Morph        bool
}

func (c shadowPipelineConfig) Hash() Hash {
	h := HashFor[shadowPipelineConfig]()
	h.Pointer(c.VertexLayout)
	h.Bool(c.Skinned)
	h.Bool(c.Morph)
	return h
}

func (c shadowPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	var values ShaderValues

	buffers := meshVertexBuffers(c.VertexLayout, &values)

	values.Set("SKINNED", c.Skinned)
	values.Set("MORPH", c.Morph)

	mod := ctx.Shader("shadow", shadowShader, values)

	return RenderPipelineDescriptor{
		Label: "shadow pipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			// a shadow view must not bind the shadow maps it renders into
			MeshViewBindGroupLayout(MeshViewBindGroupLayoutOptions{}),
			MeshBindGroupLayout,
		},
		Vertex: wgpu.VertexState{
			Module:     mod,
			EntryPoint: "vs_main",
			Buffers:    buffers,
		},
		Primitive: wgpu.PrimitiveState{
			Topology: wgpu.PrimitiveTopologyTriangleList,
			// no culling: single sided meshes like planes must cast shadows from both sides
			CullMode: wgpu.CullModeNone,
		},
		Multisample: multisampleState(1),
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormatDepth32Float,
			DepthWriteEnabled: wgpu.OptionalBoolTrue,
			DepthCompare:      wgpu.CompareFunctionGreater,
		},
	}
}

// meshVertexBuffers returns the instance and per vertex buffer layouts of a mesh
// and enables the shader values for all vertex attributes in the layout.
func meshVertexBuffers(layout *VertexLayout, values *ShaderValues) []wgpu.VertexBufferLayout {
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
		ArrayStride: uint64(layout.Size()),
		StepMode:    wgpu.VertexStepModeVertex,
	}

	for _, attr := range layout.Attributes() {
		vblPerVertex.Attributes = append(
			vblPerVertex.Attributes,
			perVertexAttrs.AtLoc(attr.Location, attr.Format),
		)

		// define the key for the shader to know about it
		key := strings.ToUpper(attr.Name)
		values.Set("MESH3D_VERTEX_ATTRIBUTES_"+key, true)
	}

	return []wgpu.VertexBufferLayout{
		vblInstances,
		vblPerVertex,
	}
}
