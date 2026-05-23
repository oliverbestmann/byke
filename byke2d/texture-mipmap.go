package byke2d

import (
	_ "embed"
	"math/bits"

	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func mipmapLevelCount(width, height uint32) uint32 {
	return uint32(bits.Len32(max(width, height)))
}

type mipmapGenerator struct {
	cache   *PipelineCache
	context *RenderContext
}

func makeMipmapGenerator(ctx *RenderContext, pipelineCache *PipelineCache) *mipmapGenerator {
	return &mipmapGenerator{
		context: ctx,
		cache:   pipelineCache,
	}
}

func (m *mipmapGenerator) Generate(texture *Texture) {
	if texture.Descriptor.MipLevelCount <= 1 || texture.Descriptor.SampleCount != 1 {
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
	inView := texture.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "Texture.MipMap.In",
		Format:          texture.Descriptor.Format,
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
		BaseMipLevel:    level,
		MipLevelCount:   1,
		BaseArrayLayer:  0,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})

	defer outView.Release()

	pipeline := m.cache.Specialize(blitConfig{
		Format: texture.Descriptor.Format,
	})

	inSampler := m.context.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "Mipmap",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeNearest,
	})

	blitTexture(m.context, enc, pipeline, inSampler, inView, outView)
}
