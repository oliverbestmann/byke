package byke2d

import (
	_ "embed"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/byke/internal/refl"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Bloom]()

//go:embed bloom.wgsl
var bloomShader string

type Bloom struct {
	byke.Component[Bloom]

	Intensity                  float32
	LowFrequencyBoost          float32
	LowFrequencyBoostCurvature float32
	HighPassFrequency          float32
	MaxMipDimension            uint32
	Scale                      glm.Vec2f
}

var BloomNatural = Bloom{
	Intensity:                  0.15,
	LowFrequencyBoost:          0.7,
	LowFrequencyBoostCurvature: 0.95,
	HighPassFrequency:          1.0,
	MaxMipDimension:            512,
	Scale:                      glm.Vec2f{1, 1},
}

type bloomPipelineConfig struct {
	TargetFormat    wgpu.TextureFormat
	FirstDownsample bool
	Upsample        bool
	UniformScale    bool
}

func (b bloomPipelineConfig) Specialize(ctx *wgpu.Device) *wgpu.RenderPipeline {
	defs := pre.Values{}
	defs.Define("UNIFORM_SCALE", b.UniformScale)
	defs.Define("FIRST_DOWNSAMPLE", b.FirstDownsample)

	var entry string
	switch {
	case b.Upsample:
		entry = "upsample"
	case b.FirstDownsample:
		entry = "downsample_first"
	default:
		entry = "downsample"
	}

	shaderCode, err := pre.Process(bloomShader, defs)
	if err != nil {
		panic(fmt.Errorf("process bloom shader: %w", err))
	}

	module := ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "Bloom Shader",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: shaderCode},
	})

	vertexState, primitiveState := prepareFullscreenShader(ctx)

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

	return ctx.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:     fmt.Sprintf("%+v", b),
		Vertex:    vertexState,
		Primitive: primitiveState,
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
		Multisample: wgpu.MultisampleState{
			Count: 1,
			Mask:  0xffffffff,
		},
	})
}

func applyBloomSystem(
	ctx *RenderContext,
	camera *CurrentCamera,
	bloomQuery byke.Query[Bloom],
	bloomPipeline Pipelines[bloomPipelineConfig],
	uniforms *ComponentUniforms[bloomUniforms],
	textureCache *TextureCache,
) {
	bloom, ok := bloomQuery.Get(camera.Entity)
	if !ok || bloom.Intensity == 0 {
		return
	}

	defer puffin.NewScope("byke2d.Bloom").End()

	isUniformScale := bloom.Scale == glm.Vec2f{1, 1}

	downsample0 := bloomPipeline.Specialize(bloomPipelineConfig{
		TargetFormat:    wgpu.TextureFormatRG11B10Ufloat,
		UniformScale:    isUniformScale,
		FirstDownsample: true,
	})

	downsampleN := bloomPipeline.Specialize(bloomPipelineConfig{
		TargetFormat: wgpu.TextureFormatRG11B10Ufloat,
		UniformScale: isUniformScale,
	})

	upsample := bloomPipeline.Specialize(bloomPipelineConfig{
		TargetFormat: wgpu.TextureFormatRG11B10Ufloat,
		UniformScale: isUniformScale,
		Upsample:     true,
	})

	upsampleT := bloomPipeline.Specialize(bloomPipelineConfig{
		TargetFormat: camera.ViewTarget.Format,
		UniformScale: isUniformScale,
		Upsample:     true,
	})

	bloomTexture := bloomGetTexture(textureCache, bloom, camera.ViewTarget.Size)

	uniforms.Write(bloomUniforms{
		Viewport: glm.Vec4f{0, 0, 1, 1},
		Scale:    bloom.Scale,
		Aspect:   camera.ViewTarget.Size[0] / camera.ViewTarget.Size[1],
	})

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Bloom"})
	defer enc.Release()

	// downsample from screen to our texture
	bloomDownsample(ctx, enc, downsample0, camera.ViewTarget.UnsampledTexture(), bloomTexture.Get(0), uniforms)

	// downsample mip levels
	for level := uint32(1); level < bloomTexture.MipCount; level++ {
		bloomDownsample(ctx, enc, downsampleN, bloomTexture.Get(level-1), bloomTexture.Get(level), uniforms)
	}

	// now upsample in reverse
	for level := bloomTexture.MipCount - 1; level >= 1; level-- {
		source := bloomTexture.Get(level)
		target := bloomTexture.Get(level - 1)
		bloomUpsample(ctx, enc, upsample, bloom, source, target, uniforms, level, bloomTexture.MipCount)
	}

	// final upsample step, render to main texture
	source := bloomTexture.Get(0)
	bloomUpsample(ctx, enc, upsampleT, bloom, source, camera.ViewTarget.UnsampledTexture(), uniforms, 0, bloomTexture.MipCount)

	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "Bloom"})
	ctx.Submit(buf)

	_ = upsample

}

func bloomDownsample(ctx *RenderContext, enc *wgpu.CommandEncoder, pipeline wx.CachedPipeline, source, target *wgpu.TextureView, uniforms *ComponentUniforms[bloomUniforms]) {
	bloomSampler := wx.CachedSampler(ctx.Device, wgpu.SamplerDescriptor{
		Label:         "Bloom Sampler",
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		AddressModeW:  wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		LodMinClamp:   0,
		LodMaxClamp:   32,
		MaxAnisotropy: 1,
	})

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Bloom Bindgroup",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: Sequential(
			BindingTextureView(source),
			BindingSampler(bloomSampler),
			uniforms.Binding(),
		),
	})

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Bloom Downsample",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:       target,
				LoadOp:     wgpu.LoadOpClear,
				StoreOp:    wgpu.StoreOpStore,
				ClearValue: wgpu.Color{},
			},
		},
	})

	pass.SetPipeline(pipeline.Pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)

	pass.End()
}

func bloomUpsample(ctx *RenderContext, enc *wgpu.CommandEncoder, pipeline wx.CachedPipeline, bloom Bloom, source, target *wgpu.TextureView, uniforms *ComponentUniforms[bloomUniforms], mip, mipCount uint32) {
	bloomSampler := wx.CachedSampler(ctx.Device, wgpu.SamplerDescriptor{
		Label:         "Bloom Sampler",
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		AddressModeW:  wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		LodMinClamp:   0,
		LodMaxClamp:   32,
		MaxAnisotropy: 1,
	})

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Bloom Bindgroup",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: Sequential(
			BindingTextureView(source),
			BindingSampler(bloomSampler),
			uniforms.Binding(),
		),
	})

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Bloom Upsample",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:       target,
				LoadOp:     wgpu.LoadOpLoad,
				StoreOp:    wgpu.StoreOpStore,
				ClearValue: wgpu.Color{},
			},
		},
	})

	bf := float64(bloomComputeBlendFactor(bloom, float32(mip), float32(mipCount-1)))

	pass.SetPipeline(pipeline.Pipeline)
	pass.SetBlendConstant(&wgpu.Color{R: bf, G: bf, B: bf, A: 1.0})
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)

	pass.End()
}

func bloomGetTexture(cache *TextureCache, bloom Bloom, cameraSize glm.Vec2f) bloomTexture {
	mipCount := uint32(max(2, bits.Len32(bloom.MaxMipDimension)) - 1)

	var mipHeightRatio float32
	if h := cameraSize[1]; h != 0 {
		mipHeightRatio = float32(bloom.MaxMipDimension) / h
	}

	texture := cache.Allocate(&wgpu.TextureDescriptor{
		Label:     "Bloom Texture",
		Usage:     wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
		Dimension: wgpu.TextureDimension2D,
		Size: wgpu.Extent3D{
			Width:              max(1, uint32(cameraSize[0]*mipHeightRatio+0.5)),
			Height:             max(1, uint32(cameraSize[0]*mipHeightRatio+0.5)),
			DepthOrArrayLayers: 1,
		},
		Format:        wgpu.TextureFormatRG11B10Ufloat,
		MipLevelCount: mipCount,
		SampleCount:   1,
	})

	return bloomTexture{
		Texture:  texture,
		MipCount: mipCount,
	}
}

type bloomTexture struct {
	MipCount uint32
	Texture  *Texture
}

func (b bloomTexture) Get(level uint32) *wgpu.TextureView {
	return b.Texture.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "Bloom Texture View",
		Format:          b.Texture.Descriptor.Format,
		BaseMipLevel:    level,
		MipLevelCount:   1,
		BaseArrayLayer:  0,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})
}

type bloomUniforms struct {
	byke.Component[bloomUniforms]

	Viewport glm.Vec4f
	Scale    glm.Vec2f
	Aspect   float32
}

func (b bloomUniforms) ToWGPU() []byte {
	var w wx.StructWriter
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

type WGPUComponent[C byke.IsComponent[C]] interface {
	byke.IsComponent[C]
	comparable
	ToWGPU() []byte
}

type ComponentUniforms[C WGPUComponent[C]] struct {
	_   byke.NoCopy
	ctx *RenderContext

	value C

	buffer     *wgpu.Buffer
	bufferSize int
}

func (c *ComponentUniforms[C]) Binding() wgpu.BindGroupEntry {
	return BindingBufferSize(c.buffer, 0, uint64(c.bufferSize))
}

func (c *ComponentUniforms[C]) Write(value C) {
	if c.value == value && c.buffer != nil {
		return
	}

	bytes := value.ToWGPU()
	if c.bufferSize >= len(bytes) {
		// re-use buffer, just update
		c.ctx.WriteBuffer(c.buffer, 0, bytes)
		return
	}

	// free existing buffer if any
	if c.buffer != nil {
		c.buffer.Release()
		c.buffer = nil
	}

	c.bufferSize = len(bytes)

	// allocate new buffer
	c.buffer = c.ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    fmt.Sprintf("Uniform Buffer %T", value),
		Contents: bytes,
		Usage:    wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
}

func (*ComponentUniforms[C]) newState(world *byke.World, _ uniformsT) byke.SystemParamState {
	return &componentUniformsSystemParamState[C]{World: world}
}

type uniformsT interface {
	newState(world *byke.World, _ uniformsT) byke.SystemParamState
}

func makeComponentUniformsSystemParamState(world *byke.World, pType reflect.Type) byke.SystemParamState {
	if !refl.ImplementsInterfaceDirectly[uniformsT](pType) {
		return nil
	}

	// pType is *ComponentUniforms[C]
	p := reflect.New(pType.Elem()).Interface().(uniformsT)
	return p.newState(world, p)
}

type componentUniformsSystemParamState[C WGPUComponent[C]] struct {
	World    *byke.World
	instance reflect.Value
}

func (p *componentUniformsSystemParamState[C]) GetValue(byke.SystemContext) (reflect.Value, error) {
	if !p.instance.IsValid() {
		ctx, ok := byke.ResourceOf[RenderContext](p.World)
		if !ok {
			return reflect.Value{}, errors.New("no RenderContext in World")
		}

		uniforms := &ComponentUniforms[C]{ctx: ctx}
		p.instance = reflect.ValueOf(uniforms)
	}

	return p.instance, nil
}

func (p *componentUniformsSystemParamState[C]) ValueType() reflect.Type {
	return reflect.TypeFor[*ComponentUniforms[C]]()
}

func (p *componentUniformsSystemParamState[C]) CleanupValue() {
}
