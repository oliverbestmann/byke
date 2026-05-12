package byke2d

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type RenderContext struct {
	Metrics         RenderMetrics
	MipmapGenerator *mipmapGenerator

	// cache for samplers
	samplerCache *lru.Cache[wgpu.SamplerDescriptor, *wgpu.Sampler]

	*wx.Context
}

func (rc *RenderContext) init(ctx *wx.Context, preCompiler *pre.Compiler) {
	rc.Context = ctx
	rc.MipmapGenerator = makeMipmapGenerator(rc, preCompiler)
	rc.samplerCache, _ = lru.NewWithEvict[wgpu.SamplerDescriptor, *wgpu.Sampler](16, samplerCacheOnEvict)
}

// CreateSampler returns a sampler matching your description. The sampler is cached,
// you must NOT call release the returned wgpu.Sampler.
func (rc *RenderContext) CreateSampler(desc wgpu.SamplerDescriptor) *wgpu.Sampler {
	if desc.MaxAnisotropy == 0 {
		// must be at least 1.
		desc.MaxAnisotropy = 1
	}

	if desc.LodMaxClamp == 0 {
		// default from wgpu
		desc.LodMaxClamp = 32
	}

	cachedSampler, ok := rc.samplerCache.Get(desc)
	if ok {
		return cachedSampler
	}

	// create a new sampler
	sampler := rc.Context.CreateSampler(new(desc))

	// and cache it for the next access
	rc.samplerCache.Add(desc, sampler)

	return sampler
}

func (rc *RenderContext) Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex {
	rc.Metrics.Submit += 1
	return rc.Context.Submit(commandBuffers...)
}

func (rc *RenderContext) TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error) {
	rc.Metrics.WriteBuffer += 1
	return rc.Context.TryWriteBuffer(buffer, offset, data)
}

func (rc *RenderContext) TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error) {
	rc.Metrics.WriteTexture += 1
	return rc.Context.TryWriteTexture(destination, data, dataLayout, writeSize)
}

func (rc *RenderContext) WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte) {
	rc.Metrics.WriteBuffer += 1
	rc.Context.WriteBuffer(buffer, bufferOffset, data)
}

func (rc *RenderContext) WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) {
	rc.Metrics.WriteTexture += 1
	rc.Context.WriteTexture(destination, data, dataLayout, writeSize)
}

func (rc *RenderContext) CreateRenderPipeline(desc *wgpu.RenderPipelineDescriptor) *wgpu.RenderPipeline {
	rc.Metrics.CreateRenderPipeline += 1
	return rc.Context.CreateRenderPipeline(desc)
}

func (rc *RenderContext) CreateBindGroup(desc *wgpu.BindGroupDescriptor) *wgpu.BindGroup {
	rc.Metrics.CreateBindGroup += 1
	return rc.Context.CreateBindGroup(desc)
}

func (rc *RenderContext) CreateShaderModule(desc *wgpu.ShaderModuleDescriptor) *wgpu.ShaderModule {
	rc.Metrics.CreateShaderModule += 1
	return rc.Context.CreateShaderModule(desc)
}

func (rc *RenderContext) Create(desc *wgpu.BindGroupLayoutDescriptor) *wgpu.BindGroupLayout {
	rc.Metrics.CreateBindGroupLayout += 1
	return rc.Context.CreateBindGroupLayout(desc)
}

func (rc *RenderContext) CreateCommandEncoder(desc *wgpu.CommandEncoderDescriptor) *wgpu.CommandEncoder {
	rc.Metrics.CreateCommandEncoder += 1
	return rc.Context.CreateCommandEncoder(desc)
}

func samplerCacheOnEvict(_ wgpu.SamplerDescriptor, value *wgpu.Sampler) {
	value.Release()
}
