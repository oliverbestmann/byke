package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed blit.wgsl
var blitShader string

type blitConfig struct {
	Format     wgpu.TextureFormat
	AlphaBlend bool
}

func (b blitConfig) EqualTo(other PipelineConfig) bool {
	return b == other
}

func (b blitConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	shader := ctx.Shader("Blit", blitShader, nil)

	blend := wgpu.BlendStateReplace
	if b.AlphaBlend {
		blend = wgpu.BlendStateAlphaBlending
	}

	return RenderPipelineDescriptor{
		Label: "Blit",

		Layout: []wgpu.BindGroupLayoutDescriptor{
			SequentialLayoutWithLabel(
				"Blit",
				BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
				BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
			),
		},

		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    b.Format,
					Blend:     &blend,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},

		Vertex:      FullscreenShaderVertexState(shader),
		Multisample: multisampleStateOne,
	}
}

func blitTextureSimple(
	ctx *RenderContext,
	pipeline Pipeline,
	sourceView, targetView *wgpu.TextureView,
) {
	sampler := ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "Blit",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeNearest,
		MinFilter:    wgpu.FilterModeNearest,
		MipmapFilter: wgpu.MipmapFilterModeNearest,
	})

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Blit"})
	defer enc.Release()

	blitTexture(ctx, enc, pipeline, sampler, sourceView, targetView)

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "Blit"})
	defer buf.Release()

	ctx.Submit(buf)
}

func blitTexture(ctx *RenderContext, enc *CommandEncoder, pipeline Pipeline, sampler *wgpu.Sampler, sourceView, targetView *wgpu.TextureView) {
	defer puffin.NewScope("byke2d.blitTexture").End()

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Blit",
		Layout: pipeline.BindGroupLayout(0),
		Entries: Sequential(
			BindingTextureView(sourceView),
			BindingSampler(sampler),
		),
	})

	defer bindGroup.Release()

	desc := &wgpu.RenderPassDescriptor{
		Label: "Blit",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:    targetView,
				LoadOp:  wgpu.LoadOpLoad,
				StoreOp: wgpu.StoreOpStore,
			},
		},
	}

	pass := enc.BeginRenderPass(desc)
	defer pass.Release()

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)
	pass.End()
}
