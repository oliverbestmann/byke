package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed fullscreen_vertex.wgsl
var blitShaderVertex string

//go:embed blit.wgsl
var blitShaderFragment string

func blitTexture(ctx *RenderContext, sourceView, targetView *wgpu.TextureView, targetFormat wgpu.TextureFormat) {
	modVertex := ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "FullscreenShaderVertex",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: blitShaderVertex},
	})

	defer modVertex.Release()

	modFragment := ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "FullscreenShaderFragment",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: blitShaderFragment},
	})

	defer modFragment.Release()

	pipeline := ctx.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Blit",
		Vertex: wgpu.VertexState{
			Module:     modVertex,
			EntryPoint: "fullscreen_vertex_shader",
		},
		Fragment: &wgpu.FragmentState{
			Module:     modFragment,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    targetFormat,
					Blend:     &wgpu.BlendStateReplace,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology: wgpu.PrimitiveTopologyTriangleList,
			CullMode: wgpu.CullModeNone,
		},
		Multisample: wgpu.MultisampleState{
			Count: 1,
			Mask:  0xffffffff,
		},
	})

	defer pipeline.Release()

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Blit",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: []wgpu.BindGroupEntry{
			{
				Binding:     0,
				TextureView: sourceView,
			},
			{
				Binding: 1,
				Sampler: wx.CachedSampler(ctx.Device, wgpu.SamplerDescriptor{
					Label:         "Blit",
					AddressModeU:  wgpu.AddressModeClampToEdge,
					AddressModeV:  wgpu.AddressModeClampToEdge,
					AddressModeW:  wgpu.AddressModeClampToEdge,
					MagFilter:     wgpu.FilterModeNearest,
					MinFilter:     wgpu.FilterModeNearest,
					MipmapFilter:  wgpu.MipmapFilterModeNearest,
					LodMinClamp:   0,
					LodMaxClamp:   32,
					MaxAnisotropy: 1,
				}),
			},
		},
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

	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)
	pass.End()

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "ResolveTexture"})
	defer buf.Release()

	ctx.Submit(buf)
}
