package byke2d

import (
	_ "embed"
	"math/bits"

	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed texture-mipmap.wgsl
var mipmapShader string

func mipmapLevelCount(width, height uint32) uint32 {
	return uint32(bits.Len32(max(width, height)))
}

type mipmapGenerator struct {
	context *wx.Context
	cache   *wx.PipelineCache[mipmapPipelineConfig]
	module  *wgpu.ShaderModule
}

func makeMipmapGenerator(ctx *wx.Context) *mipmapGenerator {
	module := ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "Texture.MipMap.Shader",
		WGSLSource: &wgpu.ShaderSourceWGSL{
			Code: mipmapShader,
		},
	})

	return &mipmapGenerator{
		context: ctx,
		cache:   wx.NewPipelineCache[mipmapPipelineConfig](ctx),
		module:  module,
	}
}

type mipmapPipelineConfig struct {
	Module      *wgpu.ShaderModule
	Format      wgpu.TextureFormat
	SampleCount uint32
}

func (m mipmapPipelineConfig) Specialize(def *wgpu.Device) *wgpu.RenderPipeline {
	return def.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "Texture.MipMap.Pipeline",
		Vertex: wgpu.VertexState{
			Module:     m.Module,
			EntryPoint: "vs_main",
		},
		Primitive: wgpu.PrimitiveState{
			Topology:         wgpu.PrimitiveTopologyTriangleStrip,
			StripIndexFormat: wgpu.IndexFormatUint16,
		},
		Multisample: wgpu.MultisampleState{
			Count: m.SampleCount,
			Mask:  0xffffffff,
		},
		Fragment: &wgpu.FragmentState{
			Module:     m.Module,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    m.Format,
					Blend:     &wgpu.BlendStateReplace,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},
	})
}

func (m *mipmapGenerator) Generate(texture *Texture) {
	if texture.Descriptor.MipLevelCount <= 1 {
		return
	}

	defer puffin.NewScopeWithValue("texture.GenerateMipMaps", texture.Descriptor.Label).End()

	enc := m.context.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{
		Label: "Texture.MipMap.Encoder",
	})

	defer enc.Release()

	for level := uint32(1); level < texture.Descriptor.MipLevelCount; level++ {
		m.generateLevel(enc, texture, level)
	}

	buf := enc.Finish(nil)
	m.context.Submit(buf)
}

func (m *mipmapGenerator) generateLevel(enc *wgpu.CommandEncoder, texture *Texture, level uint32) {
	inSampler := wx.CachedSampler(m.context.Device, wgpu.SamplerDescriptor{
		Label:         "Texture.MipMap.Sampler",
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		AddressModeW:  wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeNearest,
		LodMinClamp:   0,
		LodMaxClamp:   32,
		MaxAnisotropy: 1,
	})

	inView := texture.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "Texture.MipMap.In",
		Format:          texture.Descriptor.Format,
		Dimension:       wgpu.TextureViewDimension2D,
		BaseMipLevel:    level - 1,
		MipLevelCount:   1,
		BaseArrayLayer:  0,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})

	defer inView.Release()

	outView := texture.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "Texture.MipMap.Out",
		Format:          texture.Descriptor.Format,
		Dimension:       wgpu.TextureViewDimension2D,
		BaseMipLevel:    level,
		MipLevelCount:   1,
		BaseArrayLayer:  0,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})

	defer outView.Release()

	pipeline := m.cache.Get(mipmapPipelineConfig{
		Module:      m.module,
		Format:      texture.Descriptor.Format,
		SampleCount: texture.Descriptor.SampleCount,
	})

	bindGroup := m.context.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Texture.MipMap.BindGroup",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: []wgpu.BindGroupEntry{
			{
				Binding:     0,
				TextureView: inView,
			},
			{
				Binding: 1,
				Sampler: inSampler,
			},
		},
	})

	defer bindGroup.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Texture.MipMap.RenderPass",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:    outView,
				LoadOp:  wgpu.LoadOpClear,
				StoreOp: wgpu.StoreOpStore,
			},
		},
	})

	defer pass.Release()

	pass.SetPipeline(pipeline.Pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(6, 1, 0, 0)
	pass.End()

}
