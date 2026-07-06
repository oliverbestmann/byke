package byke2d

import (
	"errors"
	"log/slog"
	"os"
	"slices"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var forceFallbackAdapter = os.Getenv("WGPU_FORCE_FALLBACK_ADAPTER") == "1"

// RenderContext manages GPU resource creation, caching, and metrics tracking.
// It provides convenient wrappers around WebGPU objects with automatic caching of samplers
// and bind group layouts to reduce GPU memory fragmentation and creation overhead.
type RenderContext struct {
	_ byke.NoCopy

	// Metrics tracks GPU operation counts for performance analysis.
	Metrics RenderMetrics

	// MipmapGenerator creates mipmaps for textures.
	MipmapGenerator *mipmapGenerator

	// cache for samplers
	samplerCache *lru.Cache[wgpu.SamplerDescriptor, *wgpu.Sampler]

	// bindGroupLayoutCache stores bind group layouts for reuse
	bindGroupLayoutCache []cachedBindGroupLayout

	*wgpuContext
}

func (rc *RenderContext) init(world *byke.World, ctx *wgpuContext) {
	rc.wgpuContext = ctx

	pipelineCache := world.RequireResourceOf[PipelineCache]()
	rc.MipmapGenerator = makeMipmapGenerator(rc, pipelineCache)

	rc.samplerCache, _ = lru.New[wgpu.SamplerDescriptor, *wgpu.Sampler](256)
}

// CreateSampler returns a sampler for the given descriptor. The sampler is cached internally;
// do not release the returned sampler as it may be reused by other parts of the renderer.
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
	sampler := rc.wgpuContext.CreateSampler(new(desc))

	// and cache it for the next access
	rc.samplerCache.Add(desc, wgpu.Share(sampler))

	return sampler
}

// CreateBindGroupLayout returns a bind group layout for the given descriptor. Layouts are
// cached and reused across multiple bind groups with identical structure.
func (rc *RenderContext) CreateBindGroupLayout(desc wgpu.BindGroupLayoutDescriptor) *wgpu.BindGroupLayout {
	for _, cached := range rc.bindGroupLayoutCache {
		if cached.Matches(desc) {
			return cached.BindGroupLayout
		}
	}

	slog.Debug("Create BindGroupLayout", slog.String("label", desc.Label))
	bindGroupLayout := wgpu.Share(rc.wgpuContext.CreateBindGroupLayout(&desc))

	rc.bindGroupLayoutCache = append(rc.bindGroupLayoutCache, cachedBindGroupLayout{
		Descriptor:      desc,
		BindGroupLayout: bindGroupLayout,
	})

	return bindGroupLayout
}

// Submit sends the given command buffers to the GPU queue for execution.
func (rc *RenderContext) Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex {
	rc.Metrics.Submit += 1
	return rc.wgpuContext.Submit(commandBuffers...)
}

// TryWriteBuffer attempts to write data to the given buffer at the specified offset.
// Returns an error if the write fails.
func (rc *RenderContext) TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error) {
	rc.Metrics.WriteBuffer += 1
	return rc.wgpuContext.TryWriteBuffer(buffer, offset, data)
}

// TryWriteTexture attempts to write data to the given texture at the specified location.
// Returns an error if the write fails.
func (rc *RenderContext) TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error) {
	rc.Metrics.WriteTexture += 1
	return rc.wgpuContext.TryWriteTexture(destination, data, dataLayout, writeSize)
}

// WriteBuffer writes data to the given buffer at the specified offset.
// Panics if the write fails.
func (rc *RenderContext) WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte) {
	rc.Metrics.WriteBuffer += 1
	rc.wgpuContext.WriteBuffer(buffer, bufferOffset, data)
}

// WriteTexture writes data to the given texture at the specified location.
// Panics if the write fails.
func (rc *RenderContext) WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) {
	rc.Metrics.WriteTexture += 1
	rc.wgpuContext.WriteTexture(destination, data, dataLayout, writeSize)
}

// CreateRenderPipeline creates a new render pipeline for the given descriptor.
func (rc *RenderContext) CreateRenderPipeline(desc *wgpu.RenderPipelineDescriptor) *wgpu.RenderPipeline {
	rc.Metrics.CreateRenderPipeline += 1
	return rc.wgpuContext.CreateRenderPipeline(desc)
}

// CreateBindGroup creates a new bind group for the given descriptor.
func (rc *RenderContext) CreateBindGroup(desc *wgpu.BindGroupDescriptor) *wgpu.BindGroup {
	rc.Metrics.CreateBindGroup += 1
	return rc.wgpuContext.CreateBindGroup(desc)
}

// CreateShaderModule creates a new shader module for the given descriptor.
func (rc *RenderContext) CreateShaderModule(desc *wgpu.ShaderModuleDescriptor) *wgpu.ShaderModule {
	rc.Metrics.CreateShaderModule += 1
	return rc.wgpuContext.CreateShaderModule(desc)
}

// Create creates a new bind group layout for the given descriptor.
func (rc *RenderContext) Create(desc *wgpu.BindGroupLayoutDescriptor) *wgpu.BindGroupLayout {
	rc.Metrics.CreateBindGroupLayout += 1
	return rc.wgpuContext.CreateBindGroupLayout(desc)
}

// CreateCommandEncoder creates a new command encoder for recording GPU commands.
func (rc *RenderContext) CreateCommandEncoder(desc *wgpu.CommandEncoderDescriptor) *CommandEncoder {
	rc.Metrics.CreateCommandEncoder += 1
	return &CommandEncoder{
		CommandEncoder: rc.wgpuContext.CreateCommandEncoder(desc),
		metrics:        &rc.Metrics,
	}
}

// CommandEncoder wraps wgpu.CommandEncoder and tracks command recording metrics.
type CommandEncoder struct {
	*wgpu.CommandEncoder
	metrics *RenderMetrics
}

// Get creates a render pass encoder with metrics tracking.
func (c *CommandEncoder) Get(desc *wgpu.RenderPassDescriptor) *TrackedRenderPassEncoder {
	return &TrackedRenderPassEncoder{
		RenderPassEncoder: c.BeginRenderPass(desc),
		Metrics:           c.metrics,
	}
}

// Context encapsulates the low level state of the webgpu context,
// this includes the Device, Surface and active Adapter
type wgpuContext struct {
	*wgpu.Device
	*wgpu.Queue
	Instance *wgpu.Instance
	Adapter  *wgpu.Adapter
	Surface  *wgpu.Surface
}

// newContext creates a new Context for a wgpu.SurfaceDescriptor.
func newContext(sd *wgpu.SurfaceDescriptor) (st *wgpuContext, err error) {
	defer func() {
		if err != nil && st != nil {
			st.Release()
			st = nil
		}
	}()

	st = new(wgpuContext)

	// create the webgpu instance
	instance := wgpu.CreateInstance(nil)
	st.Instance = instance

	// create a Surface based on the window
	st.Surface = instance.CreateSurface(sd)

	// create an adapter that can render to the Surface
	st.Adapter, err = instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		ForceFallbackAdapter: forceFallbackAdapter,
		CompatibleSurface:    st.Surface,
	})
	if err != nil {
		return
	}

	if !st.Adapter.HasFeature(wgpu.FeatureNameRG11B10UfloatRenderable) {
		err = errors.New("missing feature RG11B10UfloatRenderable required for bloom")
		return
	}

	// get a Device with the default settings
	st.Device, err = st.Adapter.RequestDevice(&wgpu.DeviceDescriptor{
		RequiredFeatures: []wgpu.FeatureName{
			wgpu.FeatureNameRG11B10UfloatRenderable,
		},
	})
	if err != nil {
		return
	}

	// cache a reference to the queue
	st.Queue = st.Device.GetQueue()

	return st, nil
}

func (wc *wgpuContext) Release() {
	if wc.Queue != nil {
		wc.Queue.Release()
		wc.Queue = nil
	}

	if wc.Device != nil {
		wc.Device.Release()
		wc.Device = nil
	}

	if wc.Adapter != nil {
		wc.Adapter.Release()
		wc.Adapter = nil
	}

	if wc.Surface != nil {
		wc.Surface.Release()
		wc.Surface = nil
	}

	if wc.Instance != nil {
		wc.Instance.Release()
		wc.Instance = nil
	}
}

type cachedBindGroupLayout struct {
	Descriptor      wgpu.BindGroupLayoutDescriptor
	BindGroupLayout *wgpu.BindGroupLayout
}

func (c *cachedBindGroupLayout) Matches(desc wgpu.BindGroupLayoutDescriptor) bool {
	return desc.Label == c.Descriptor.Label && c.entiresAreEqual(desc)
}

func (c *cachedBindGroupLayout) entiresAreEqual(desc wgpu.BindGroupLayoutDescriptor) bool {
	return slices.EqualFunc(desc.Entries, c.Descriptor.Entries, func(lhs, rhs wgpu.BindGroupLayoutEntry) bool {
		return lhs.Binding == rhs.Binding &&
			lhs.Visibility == rhs.Visibility &&
			lhs.Buffer == rhs.Buffer &&
			lhs.Sampler == rhs.Sampler &&
			lhs.Texture == rhs.Texture &&
			lhs.StorageTexture == rhs.StorageTexture
	})
}
