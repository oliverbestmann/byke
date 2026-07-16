package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed skybox.wgsl
var skyboxShader string

var _ = byke.ValidateComponent[Skybox]()

type Skybox struct {
	byke.Component[Skybox]
	Texture   *Texture
	Intensity Color
}

func (s Skybox) ToWGPU() []byte {
	var w wgsl.StructWriter
	w.AppendVec3f(s.Intensity.ToVec3f())
	return w.Bytes()
}

func pluginSkybox(app *byke.App) {
	app.AddPlugin(ComponentUniformsPlugin[Skybox])
	app.InitResource[skyboxBindGroups]()

	app.AddSystems(Render, byke.
		System(prepareSkyboxBindGroupsSystem).
		InSet(RenderPhasePrepareBindGroups))

	app.AddSystems(Core3d, byke.
		System(drawSkyboxSystem).
		InSet(Core3dSky))
}

type skyboxPipeline struct {
	ViewFormat  wgpu.TextureFormat
	DepthFormat wgpu.TextureFormat
	SampleCount uint32
}

func (s skyboxPipeline) EqualTo(other PipelineConfig) bool {
	return s == other
}

func (s skyboxPipeline) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	shader := ctx.Shader("Skybox", skyboxShader, nil)

	return RenderPipelineDescriptor{
		Label: "Skybox",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			ViewBindGroupLayout,
			skyboxBindGroupLayout,
		},
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "skybox_vertex",
		},
		Primitive: wgpu.PrimitiveState{
			CullMode:  wgpu.CullModeNone,
			Topology:  wgpu.PrimitiveTopologyTriangleStrip,
			FrontFace: wgpu.FrontFaceCCW,
		},
		Multisample: multisampleState(s.SampleCount),
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "skybox_fragment",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    s.ViewFormat,
					Blend:     nil,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            s.DepthFormat,
			DepthWriteEnabled: wgpu.OptionalBoolFalse,
			DepthCompare:      wgpu.CompareFunctionGreaterEqual,
		},
	}
}

var skyboxBindGroupLayout = SequentialLayoutWithLabel("Skybox",
	BindingLayoutTextureCube(wgpu.TextureSampleTypeFloat, false),
	BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
)

type skyboxBindGroups struct {
	tickCache[byke.EntityId, *wgpu.BindGroup]
}

func prepareSkyboxBindGroupsSystem(
	ctx *RenderContext,
	bindGroups *skyboxBindGroups,
	uniforms *ComponentUniforms[Skybox],
	query byke.Query[struct {
		EntityId byke.EntityId
		Skybox   Skybox
		Offset   DynamicOffset[Skybox]
	}],
) {
	bindGroups.Tick()

	for item := range query.Items() {
		if _, ok := bindGroups.Get(item.EntityId); ok {
			continue
		}

		bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Skybox",
			Layout: ctx.CreateBindGroupLayout(skyboxBindGroupLayout),
			Entries: Sequential(
				BindingTextureView(item.Skybox.Texture.TextureView),
				BindingSampler(item.Skybox.Texture.Sampler),
				uniforms.Binding(),
			),
		})

		bindGroups.Add(item.EntityId, bindGroup)
	}
}

func drawSkyboxSystem(
	ctx *RenderContext,
	pipelines *PipelineCache,
	bindGroups *skyboxBindGroups,
	viewBindGroup ViewBindGroup,
	viewQuery ViewQuery[struct {
		EntityId     byke.EntityId
		SkyboxOffset DynamicOffset[Skybox]
		ViewOffset   DynamicOffset[ViewUniforms]
		Target       *ViewTarget
		DepthTexture *ViewDepthTexture
	}],
) {
	view := viewQuery.Get()

	pipeline := pipelines.Specialize(skyboxPipeline{
		ViewFormat:  view.Target.Format,
		DepthFormat: view.DepthTexture.Format,
		SampleCount: view.Target.SampleCount,
	})

	bindGroup, ok := bindGroups.Get(view.EntityId)
	if !ok {
		panic("bindGroup for skybox is missing")
	}

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Skybox"})
	defer enc.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label:                  "Skybox",
		ColorAttachments:       []wgpu.RenderPassColorAttachment{view.Target.Attachment()},
		DepthStencilAttachment: new(view.DepthTexture.ReadOnly()),
	})
	defer pass.Release()

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, viewBindGroup.BindGroup, []uint32{view.ViewOffset.Offset})
	pass.SetBindGroup(1, bindGroup, []uint32{view.SkyboxOffset.Offset})
	pass.Draw(3, 1, 0, 0)
	pass.End()

	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "Skybox"})
	defer buf.Release()

	ctx.Submit(buf)
}
