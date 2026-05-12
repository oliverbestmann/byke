package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed blit.wgsl
var blitShaderFragment string

type blitConfig struct {
	TargetFormat wgpu.TextureFormat
}

func (b blitConfig) Specialize() SpecializedPipeline {
	return SpecializedPipeline{
		ShaderLabel:    "Blit",
		Shader:         FullscreenVertexShader,
		FragmentShader: blitShaderFragment,
		Descriptor: wgpu.RenderPipelineDescriptor{
			Label:  "Blit",
			Vertex: wgpu.VertexState{EntryPoint: FullscreenShaderEntryPoint},
			Fragment: &wgpu.FragmentState{
				EntryPoint: "fs_main",
				Targets: []wgpu.ColorTargetState{
					{
						Format:    b.TargetFormat,
						Blend:     &wgpu.BlendStateReplace,
						WriteMask: wgpu.ColorWriteMaskAll,
					},
				},
			},
			Multisample: multisampleStateOne,
		},
	}
}

func blitTexture(
	ctx *RenderContext,
	pipeline Pipeline,
	sourceView, targetView *wgpu.TextureView,
) {
	defer puffin.NewScope("byke2d.blitTexture").End()

	sampler := ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "Blit",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeNearest,
		MinFilter:    wgpu.FilterModeNearest,
		MipmapFilter: wgpu.MipmapFilterModeNearest,
	})
	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Blit",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: Sequential(
			BindingTextureView(sourceView),
			BindingSampler(sampler),
		),
	})

	defer bindGroup.Release()

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "BlitTexture"})
	defer enc.Release()

	desc := &wgpu.RenderPassDescriptor{
		Label: "BlitTexture",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:    targetView,
				LoadOp:  wgpu.LoadOpClear,
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

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "ResolveTexture"})
	defer buf.Release()

	ctx.Submit(buf)
}
