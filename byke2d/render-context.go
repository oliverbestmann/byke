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

type RenderContext struct {
	_ byke.NoCopy

	Metrics         RenderMetrics
	MipmapGenerator *mipmapGenerator

	// cache for samplers
	samplerCache         *lru.Cache[wgpu.SamplerDescriptor, *wgpu.Sampler]
	bindGroupLayoutCache []cachedBindGroupLayout

	*wgpuContext
}

func (rc *RenderContext) init(world *byke.World, ctx *wgpuContext) {
	rc.wgpuContext = ctx

	pipelineCache := byke.RequireResourceOf[PipelineCache](world)
	rc.MipmapGenerator = makeMipmapGenerator(rc, pipelineCache)

	rc.samplerCache, _ = lru.New[wgpu.SamplerDescriptor, *wgpu.Sampler](256)
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
	sampler := rc.wgpuContext.CreateSampler(new(desc))

	// and cache it for the next access
	rc.samplerCache.Add(desc, wgpu.Share(sampler))

	return sampler
}

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

func (rc *RenderContext) Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex {
	rc.Metrics.Submit += 1
	return rc.wgpuContext.Submit(commandBuffers...)
}

func (rc *RenderContext) TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error) {
	rc.Metrics.WriteBuffer += 1
	return rc.wgpuContext.TryWriteBuffer(buffer, offset, data)
}

func (rc *RenderContext) TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error) {
	rc.Metrics.WriteTexture += 1
	return rc.wgpuContext.TryWriteTexture(destination, data, dataLayout, writeSize)
}

func (rc *RenderContext) WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte) {
	rc.Metrics.WriteBuffer += 1
	rc.wgpuContext.WriteBuffer(buffer, bufferOffset, data)
}

func (rc *RenderContext) WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) {
	rc.Metrics.WriteTexture += 1
	rc.wgpuContext.WriteTexture(destination, data, dataLayout, writeSize)
}

func (rc *RenderContext) CreateRenderPipeline(desc *wgpu.RenderPipelineDescriptor) *wgpu.RenderPipeline {
	rc.Metrics.CreateRenderPipeline += 1
	return rc.wgpuContext.CreateRenderPipeline(desc)
}

func (rc *RenderContext) CreateBindGroup(desc *wgpu.BindGroupDescriptor) *wgpu.BindGroup {
	rc.Metrics.CreateBindGroup += 1
	return rc.wgpuContext.CreateBindGroup(desc)
}

func (rc *RenderContext) CreateShaderModule(desc *wgpu.ShaderModuleDescriptor) *wgpu.ShaderModule {
	rc.Metrics.CreateShaderModule += 1
	return rc.wgpuContext.CreateShaderModule(desc)
}

func (rc *RenderContext) Create(desc *wgpu.BindGroupLayoutDescriptor) *wgpu.BindGroupLayout {
	rc.Metrics.CreateBindGroupLayout += 1
	return rc.wgpuContext.CreateBindGroupLayout(desc)
}

func (rc *RenderContext) CreateCommandEncoder(desc *wgpu.CommandEncoderDescriptor) *wgpu.CommandEncoder {
	rc.Metrics.CreateCommandEncoder += 1
	return rc.wgpuContext.CreateCommandEncoder(desc)
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
