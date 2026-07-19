package byke2d

import (
	_ "embed"
	"fmt"
	"math"
	"math/bits"
	"unsafe"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var (
	_ = byke.ValidateComponent[Bloom]()
)

//go:embed bloom.wgsl
var bloomShader string

func pluginBloom(app *byke.App) {
	app.InitResource[bloomCache]()

	app.AddPlugin(ComponentUniformsPlugin[bloomUniforms])

	app.AddSystems(Render, byke.
		System(prepareBloomUniformsSystem).
		Chain().
		InSet(RenderPhaseQueue))

	app.AddSystems(Render, byke.
		System(prepareBloomBindGroupsSystem).
		Chain().
		InSet(RenderPhasePrepareBindGroups))

	app.AddSystems(Core2d, byke.
		System(applyBloomSystem).
		Before(tonemappingSystem).
		Chain().
		InSet(Core2dPostProcessing))

	app.AddSystems(Core3d, byke.
		System(applyBloomSystem).
		Before(tonemappingSystem).
		Chain().
		InSet(Core3dPostProcessing))

}

type Bloom struct {
	byke.Component[Bloom]

	Intensity                  float32
	LowFrequencyBoost          float32
	LowFrequencyBoostCurvature float32
	HighPassFrequency          float32
	MaxMipDimension            uint32
	Scale                      glm.Vec2f
}

func (b Bloom) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		bloomUniforms{},
	}
}

var BloomNatural = Bloom{
	Intensity:                  0.15,
	LowFrequencyBoost:          0.7,
	LowFrequencyBoostCurvature: 0.95,
	HighPassFrequency:          1.0,
	MaxMipDimension:            512,
	Scale:                      glm.Vec2f{1, 1},
}

var bloomBindGroupLayout = SequentialLayout(
	BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
	BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
)

type bloomPipelineConfig struct {
	TargetFormat    wgpu.TextureFormat
	FirstDownsample bool
	Upsample        bool
	UniformScale    bool
}

func (b bloomPipelineConfig) EqualTo(other PipelineConfig) bool {
	return b == other
}

func (b bloomPipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	values := ShaderValues{}
	values.Define("UNIFORM_SCALE", b.UniformScale)
	values.Define("FIRST_DOWNSAMPLE", b.FirstDownsample)

	var entry string
	switch {
	case b.Upsample:
		entry = "upsample"
	case b.FirstDownsample:
		entry = "downsample_first"
	default:
		entry = "downsample"
	}

	var blend *wgpu.BlendState
	if b.Upsample {
		blend = &wgpu.BlendState{
			Color: wgpu.BlendComponent{
				SrcFactor: wgpu.BlendFactorConstant,
				DstFactor: wgpu.BlendFactorOneMinusConstant,
				Operation: wgpu.BlendOperationAdd,
			},
			Alpha: wgpu.BlendComponent{
				SrcFactor: wgpu.BlendFactorZero,
				DstFactor: wgpu.BlendFactorOne,
				Operation: wgpu.BlendOperationAdd,
			},
		}
	}

	module := ctx.Shader("Bloom", bloomShader, values)

	return RenderPipelineDescriptor{
		Label: fmt.Sprintf("%+v", b),

		Layout: []wgpu.BindGroupLayoutDescriptor{
			bloomBindGroupLayout,
		},

		Fragment: &wgpu.FragmentState{
			Module:     module,
			EntryPoint: entry,
			Targets: []wgpu.ColorTargetState{
				{
					Format:    b.TargetFormat,
					Blend:     blend,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},

		Vertex:      FullscreenShaderVertexState(module),
		Multisample: multisampleStateOne,
	}
}

func prepareBloomUniformsSystem(
	camerasQuery byke.Query[struct {
		BloomUniforms *bloomUniforms
		ViewTarget    *ViewTarget
		Bloom         Bloom
	}],
) {
	for camera := range camerasQuery.Items() {
		*camera.BloomUniforms = bloomUniforms{
			Viewport: glm.Vec4f{0, 0, 1, 1},
			Scale:    camera.Bloom.Scale,
			Aspect:   camera.ViewTarget.Size[0] / camera.ViewTarget.Size[1],
		}
	}
}

type bloomTextureKey struct {
	CameraId byke.EntityId
	Texture  unsafe.Pointer
}

type bloomBindGroup struct {
	*wgpu.BindGroup
	UniformsBufferRef *wgpu.Buffer
}

type bloomCache struct {
	byEntityId map[byke.EntityId]bloomTexture
	textures   tickCache[bloomTextureKey, bloomTexture]
	bindGroups tickCache[bloomTextureKey, bloomBindGroup]
}

type prepareBloomQueryValues struct {
	CameraId      byke.EntityId
	ViewTarget    *ViewTarget
	Bloom         Bloom
	BloomUniforms *bloomUniforms
}

func prepareBloomBindGroupsSystem(
	ctx *RenderContext,
	textureCache *TextureCache,
	uniforms *ComponentUniforms[bloomUniforms],
	bloomCache *bloomCache,
	camerasQuery byke.Query[prepareBloomQueryValues],
) {
	ensureMapIsInitialized(&bloomCache.byEntityId)
	clear(bloomCache.byEntityId)

	bloomCache.textures.Tick()
	bloomCache.bindGroups.Tick()

	for view := range camerasQuery.Items() {
		if view.Bloom.Intensity == 0 {
			continue
		}

		// get possibly cached texture
		bloomTexture := ensureCachedBloomTexture(textureCache, bloomCache, view)
		bloomCache.byEntityId[view.CameraId] = bloomTexture

		// first downsample goes from camera output
		cameraTex := view.ViewTarget.UnsampledTexture()
		ensureCachedBindGroup(ctx, bloomCache, uniforms, view, cameraTex)

		// then create views for all other sources
		for level := range bloomTexture.MipCount() {
			source := bloomTexture.Get(level)
			ensureCachedBindGroup(ctx, bloomCache, uniforms, view, source)
		}
	}
}

func ensureCachedBindGroup(
	ctx *RenderContext,
	cache *bloomCache,
	uniforms *ComponentUniforms[bloomUniforms],
	view prepareBloomQueryValues,
	source *wgpu.TextureView,
) {

	key := bloomTextureKey{
		CameraId: view.CameraId,
		Texture:  unsafe.Pointer(source),
	}

	if existing, ok := cache.bindGroups.Get(key); ok {
		if existing.UniformsBufferRef == uniforms.buffer {
			return
		}

		// not the same config, need to redo
		existing.Release()
	}

	bloomSampler := ctx.CreateSampler(BloomSamplerDescriptor)

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Bloom",
		Layout: ctx.CreateBindGroupLayout(bloomBindGroupLayout),
		Entries: Sequential(
			BindingTextureView(source),
			BindingSampler(bloomSampler),
			uniforms.Binding(),
		),
	})

	cache.bindGroups.Add(key, bloomBindGroup{
		BindGroup:         bindGroup,
		UniformsBufferRef: uniforms.buffer,
	})
}

func ensureCachedBloomTexture(textureCache *TextureCache, bloomCache *bloomCache, view prepareBloomQueryValues) bloomTexture {
	bloomTex := bloomGetTexture(textureCache, view.ViewTarget, view.Bloom)

	keyTexture := bloomTextureKey{
		CameraId: view.CameraId,
		Texture:  unsafe.Pointer(bloomTex),
	}

	bloomTexture, ok := bloomCache.textures.Get(keyTexture)
	if !ok {
		bloomTexture = bloomTextureCreate(bloomTex)
		bloomCache.textures.Add(keyTexture, bloomTexture)
	}

	return bloomTexture
}

type bloomViewQuery struct {
	CameraId            byke.EntityId
	Bloom               Bloom
	ViewTarget          *ViewTarget
	BloomUniformsOffset DynamicOffset[bloomUniforms]
}

func applyBloomSystem(
	ctx *RenderContext,
	pipelines *PipelineCache,
	bloomCache *bloomCache,
	viewQuery ViewQuery[bloomViewQuery],
) {
	view := viewQuery.Get()
	if view.Bloom.Intensity == 0 {
		return
	}

	defer puffin.NewScope("byke2d.Bloom").End()

	isUniformScale := view.Bloom.Scale == glm.Vec2f{1, 1}

	downsample0 := pipelines.Specialize(bloomPipelineConfig{
		TargetFormat:    wgpu.TextureFormatRG11B10Ufloat,
		UniformScale:    isUniformScale,
		FirstDownsample: true,
	})

	downsampleN := pipelines.Specialize(bloomPipelineConfig{
		TargetFormat: wgpu.TextureFormatRG11B10Ufloat,
		UniformScale: isUniformScale,
	})

	upsampleN := pipelines.Specialize(bloomPipelineConfig{
		TargetFormat: wgpu.TextureFormatRG11B10Ufloat,
		UniformScale: isUniformScale,
		Upsample:     true,
	})

	upsample0 := pipelines.Specialize(bloomPipelineConfig{
		TargetFormat: view.ViewTarget.Format,
		UniformScale: isUniformScale,
		Upsample:     true,
	})

	bloomTexture, ok := bloomCache.byEntityId[view.CameraId]
	if !ok {
		panic("no bloom texture found for camera")
	}

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Bloom"})
	defer enc.Release()

	// downsample from screen to our texture
	bloomDownsample(enc, bloomCache, view, downsample0, view.ViewTarget.UnsampledTexture(), bloomTexture.Get(0))

	// downsample mip levels
	for level := uint32(1); level < bloomTexture.MipCount(); level++ {
		bloomDownsample(enc, bloomCache, view, downsampleN, bloomTexture.Get(level-1), bloomTexture.Get(level))
	}

	// now upsample in reverse
	for level := bloomTexture.MipCount() - 1; level >= 1; level-- {
		source := bloomTexture.Get(level)
		target := bloomTexture.Get(level - 1)
		bloomUpsample(enc, bloomCache, view, upsampleN, source, target, level, bloomTexture.MipCount())
	}

	// final upsample step, render to main texture
	source := bloomTexture.Get(0)
	bloomUpsample(enc, bloomCache, view, upsample0, source, view.ViewTarget.UnsampledTexture(), 0, bloomTexture.MipCount())

	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "Bloom"})
	ctx.Submit(buf)
}

func bloomDownsample(
	enc *CommandEncoder,
	cache *bloomCache,
	view bloomViewQuery,
	pipeline Pipeline,
	source, target *wgpu.TextureView,
) {
	bindGroup, ok := cache.bindGroups.Get(bloomTextureKey{
		CameraId: view.CameraId,
		Texture:  unsafe.Pointer(source),
	})
	if !ok {
		panic("bindGroup for bloom pass not found")
	}

	pass := bloomPrepareRenderPass(enc, target, wgpu.LoadOpClear)

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, bindGroup.BindGroup, []uint32{view.BloomUniformsOffset.Offset})
	pass.Draw(3, 1, 0, 0)

	pass.End()
}

func bloomUpsample(
	enc *CommandEncoder,
	cache *bloomCache,
	view bloomViewQuery,
	pipeline Pipeline,
	source, target *wgpu.TextureView,
	mip,
	mipCount uint32,
) {
	bindGroup, ok := cache.bindGroups.Get(bloomTextureKey{
		CameraId: view.CameraId,
		Texture:  unsafe.Pointer(source),
	})
	if !ok {
		panic("bindGroup for bloom pass not found")
	}

	bf := float64(bloomComputeBlendFactor(view.Bloom, float32(mip), float32(mipCount-1)))

	pass := bloomPrepareRenderPass(enc, target, wgpu.LoadOpLoad)

	pass.SetPipeline(pipeline.Get())
	pass.SetBlendConstant(&wgpu.Color{R: bf, G: bf, B: bf, A: 1.0})
	pass.SetBindGroup(0, bindGroup.BindGroup, []uint32{view.BloomUniformsOffset.Offset})
	pass.Draw(3, 1, 0, 0)

	pass.End()
}

var BloomSamplerDescriptor = wgpu.SamplerDescriptor{
	Label:        "Bloom",
	AddressModeU: wgpu.AddressModeClampToEdge,
	AddressModeV: wgpu.AddressModeClampToEdge,
	AddressModeW: wgpu.AddressModeClampToEdge,
	MagFilter:    wgpu.FilterModeLinear,
	MinFilter:    wgpu.FilterModeLinear,
	MipmapFilter: wgpu.MipmapFilterModeLinear,
}

func bloomPrepareRenderPass(enc *CommandEncoder, target *wgpu.TextureView, loadOp wgpu.LoadOp) *TrackedRenderPassEncoder {
	return enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Bloom",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:    target,
				LoadOp:  loadOp,
				StoreOp: wgpu.StoreOpStore,
			},
		},
	})
}

func bloomGetTexture(texCache *TextureCache, viewTarget *ViewTarget, bloom Bloom) *Texture {
	mipCount := uint32(max(2, bits.Len32(bloom.MaxMipDimension)) - 1)

	cameraSize := viewTarget.Size

	var mipHeightRatio float32
	if h := cameraSize[1]; h != 0 {
		mipHeightRatio = float32(bloom.MaxMipDimension) / h
	}

	return texCache.Allocate(&wgpu.TextureDescriptor{
		Label:     "Bloom Texture",
		Usage:     wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
		Dimension: wgpu.TextureDimension2D,
		Size: wgpu.Extent3D{
			Width:              max(1, uint32(cameraSize[0]*mipHeightRatio+0.5)),
			Height:             max(1, uint32(cameraSize[1]*mipHeightRatio+0.5)),
			DepthOrArrayLayers: 1,
		},
		Format:        wgpu.TextureFormatRG11B10Ufloat,
		MipLevelCount: mipCount,
		SampleCount:   1,
	})
}

type bloomTexture struct {
	views []*wgpu.TextureView
}

func bloomTextureCreate(texture *Texture) bloomTexture {
	var bt bloomTexture

	for level := range texture.Descriptor.MipLevelCount {
		view := texture.Texture.CreateView(&wgpu.TextureViewDescriptor{
			Label:           fmt.Sprintf("Bloom Texture View level=%d", level),
			BaseMipLevel:    level,
			MipLevelCount:   1,
			ArrayLayerCount: 1,
		})

		bt.views = append(bt.views, view)
	}

	return bt
}

func (b bloomTexture) MipCount() uint32 {
	return uint32(len(b.views))
}

func (b bloomTexture) Get(level uint32) *wgpu.TextureView {
	return b.views[level]
}

func (b bloomTexture) Release() {
	for _, view := range b.views {
		view.Release()
	}
}

type bloomUniforms struct {
	byke.Component[bloomUniforms]

	Viewport glm.Vec4f
	Scale    glm.Vec2f
	Aspect   float32
}

func (b bloomUniforms) ToWGPU() []byte {
	var w wgsl.StructWriter
	w.AppendVec4f(b.Viewport)
	w.AppendVec2f(b.Scale)
	w.AppendFloat32(b.Aspect)
	return w.Bytes()
}

func bloomComputeBlendFactor(bloom Bloom, mip float32, maxMip float32) float32 {
	lfBoost := (1.0 - float32(math.Pow(
		float64(1.0-(mip/maxMip)),
		float64(1.0/(1.0-bloom.LowFrequencyBoostCurvature)),
	))) * bloom.LowFrequencyBoost

	lfBoost *= 1.0 - bloom.Intensity

	highPassLq := 1.0 - min(1.0, max(0.0, ((mip/maxMip)-bloom.HighPassFrequency)/bloom.HighPassFrequency))

	return (bloom.Intensity + lfBoost) * highPassLq
}
