package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed fullscreen_vertex.wgsl
var blitShaderVertex string

//go:embed blit.wgsl
var blitShaderFragment string

type blitConfig struct {
	TargetFormat wgpu.TextureFormat
}

func (b blitConfig) Specialize(def *wgpu.Device) *wgpu.RenderPipeline {
	modVertex := def.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "FullscreenShaderVertex",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: blitShaderVertex},
	})

	defer modVertex.Release()

	modFragment := def.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "FullscreenShaderFragment",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: blitShaderFragment},
	})

	defer modFragment.Release()

	return def.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
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
					Format:    b.TargetFormat,
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
}

func blitTexture(
	ctx *RenderContext,
	pipeline wx.CachedPipeline,
	sourceView, targetView *wgpu.TextureView,
) {
	defer puffin.NewScope("byke2d.blitTexture").End()

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

	pass.SetPipeline(pipeline.Pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)
	pass.End()

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "ResolveTexture"})
	defer buf.Release()

	ctx.Submit(buf)
}
