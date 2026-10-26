package main

import (
	_ "embed"

	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// A custom post-processing effect: a fullscreen pass that splits the color
// channels and darkens the corners (vignette). It runs for every 2d camera
// after the built-in post-processing (bloom and tonemapping on HDR cameras)
// and before the image is copied to the window.

//go:embed assets/post-process.wgsl
var postProcessShader string

// PostProcessSet orders our pass between the built-in post-processing and the blit.
var PostProcessSet = (&SystemSet{Name: "PostProcess"}).
	After(Core2dPostProcessing).
	Before(Core2dBlit)

func main() {
	var app App

	app.AddPlugin(PluginRender)

	app.ConfigureSystemSets(Core2d, PostProcessSet)
	app.AddSystems(Core2d, System(postProcessSystem).InSet(PostProcessSet))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands) {
	commands.Spawn(
		Camera{},
		Camera2d,
		ClearColor{Color: ColorSRGB(0.5, 0.5, 0.5)},
		OrthographicProjection{
			ViewportOrigin: glm.Vec2f{0.5, 0.5},
			ScalingMode:    ScalingModeAutoMin{MinWidth: 800, MinHeight: 600},
		},
	)

	meshes := []*Mesh{
		Circle(60, 64),
		Rectangle(glm.Vec2f{110, 110}),
		RegularPolygon(65, 6),
		Triangle(65),
	}

	// white shapes show the color split best, it is strongest at the edges of the screen
	for idx, mesh := range meshes {
		for row := range 3 {
			commands.Spawn(
				TransformFromXY(float32(idx)*180-270, float32(row)*180-180),
				MeshInstance{Mesh: mesh},
				ColorMaterial{Color: ColorWhite},
			)
		}
	}
}

func postProcessSystem(
	ctx *RenderContext,
	pipelines *PipelineCache,
	viewQuery ViewQuery[struct {
		ViewTarget *ViewTarget
	}],
) {
	view := viewQuery.Get()

	// Source holds the image rendered so far, we write the result to Target.
	pp := view.ViewTarget.PostProcess()

	pipeline := pipelines.Specialize(postProcessPipeline{Format: view.ViewTarget.Format})

	sampler := ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "PostProcess",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
	})

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "PostProcess",
		Layout: pipeline.BindGroupLayout(0),
		Entries: Sequential(
			BindingTextureView(pp.Source),
			BindingSampler(sampler),
		),
	})
	defer bindGroup.Release()

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "PostProcess"})
	defer enc.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label:            "PostProcess",
		ColorAttachments: []wgpu.RenderPassColorAttachment{pp.Target.UnsampledAttachment()},
	})

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)
	pass.End()

	ctx.Submit(enc.Finish(nil))
}

// postProcessPipeline is cached by the PipelineCache, one pipeline per target format.
type postProcessPipeline struct {
	Format wgpu.TextureFormat
}

func (p postProcessPipeline) Hash() Hash {
	h := HashFor[postProcessPipeline]()
	h.Int(p.Format)
	return h
}

func (p postProcessPipeline) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	module := ctx.Shader("PostProcess", postProcessShader, ShaderValues{})

	return RenderPipelineDescriptor{
		Label: "PostProcess",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			SequentialLayout(
				BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
				BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
			),
		},
		Vertex: FullscreenShaderVertexState(module),
		Fragment: &wgpu.FragmentState{
			Module:     module,
			EntryPoint: "fragment",
			Targets: []wgpu.ColorTargetState{
				{Format: p.Format, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xffffffff},
	}
}
