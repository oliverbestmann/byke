package byke2d

import (
	_ "embed"
	"fmt"
	"math"
	"math/bits"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Bloom]()
var _ = byke.ValidateComponent[bloomTexture]()

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

func (b Bloom) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		bloomTexture{},
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

type bloomPipelineConfig struct {
	TargetFormat    wgpu.TextureFormat
	FirstDownsample bool
	Upsample        bool
	UniformScale    bool
}

func (b bloomPipelineConfig) Specialize() SpecializedPipeline {
	values := pre.Values{}
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

	return SpecializedPipeline{
		ShaderLabel:  "Bloom",
		Shader:       bloomShader,
		ShaderValues: values,
		Descriptor: wgpu.RenderPipelineDescriptor{
			Label: fmt.Sprintf("%+v", b),
			Fragment: &wgpu.FragmentState{
				EntryPoint: entry,
				Targets: []wgpu.ColorTargetState{
					{
						Format:    b.TargetFormat,
						Blend:     blend,
						WriteMask: wgpu.ColorWriteMaskAll,
					},
				},
			},
			Vertex:      FullscreenShaderVertexState,
			Multisample: multisampleStateOne,
		},
	}
}

func prepareBloomUniforms(
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

type bloomViewQuery struct {
	Entity     byke.EntityId
	Bloom      Bloom
	Texture    bloomTexture
	ViewTarget *ViewTarget
}

func applyBloomSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	bloomPipeline Pipelines[bloomPipelineConfig],
	uniforms *ComponentUniforms[bloomUniforms],
	textureCache *TextureCache,
	viewQuery ViewQuery[bloomViewQuery],
) {
	view := viewQuery.Get()

	if view.Bloom.Intensity == 0 {
		return
	}

	defer puffin.NewScope("byke2d.Bloom").End()

	isUniformScale := view.Bloom.Scale == glm.Vec2f{1, 1}

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
		TargetFormat: view.ViewTarget.Format,
		UniformScale: isUniformScale,
		Upsample:     true,
	})

	bloomTexture := bloomGetTexture(commands, textureCache, view)

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Bloom"})
	defer enc.Release()

	// downsample from screen to our texture
	bloomDownsample(ctx, enc, downsample0, view.ViewTarget.UnsampledTexture(), bloomTexture.Get(0), uniforms)

	// downsample mip levels
	for level := uint32(1); level < bloomTexture.MipCount(); level++ {
		bloomDownsample(ctx, enc, downsampleN, bloomTexture.Get(level-1), bloomTexture.Get(level), uniforms)
	}

	// now upsample in reverse
	for level := bloomTexture.MipCount() - 1; level >= 1; level-- {
		source := bloomTexture.Get(level)
		target := bloomTexture.Get(level - 1)
		bloomUpsample(ctx, enc, upsample, view.Bloom, source, target, uniforms, level, bloomTexture.MipCount())
	}

	// final upsample step, render to main texture
	source := bloomTexture.Get(0)
	bloomUpsample(ctx, enc, upsampleT, view.Bloom, source, view.ViewTarget.UnsampledTexture(), uniforms, 0, bloomTexture.MipCount())

	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "Bloom"})
	ctx.Submit(buf)
}

func bloomDownsample(ctx *RenderContext, enc *wgpu.CommandEncoder, pipeline Pipeline, source, target *wgpu.TextureView, uniforms *ComponentUniforms[bloomUniforms]) {
	pass, bindGroup := bloomPrepareRenderPass(ctx, enc, pipeline, source, target, uniforms, "Bloom Downsample", wgpu.LoadOpClear)
	defer bindGroup.Release()

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)

	pass.End()
}

func bloomUpsample(ctx *RenderContext, enc *wgpu.CommandEncoder, pipeline Pipeline, bloom Bloom, source, target *wgpu.TextureView, uniforms *ComponentUniforms[bloomUniforms], mip, mipCount uint32) {
	pass, bindGroup := bloomPrepareRenderPass(ctx, enc, pipeline, source, target, uniforms, "Bloom Upsample", wgpu.LoadOpLoad)
	defer bindGroup.Release()

	bf := float64(bloomComputeBlendFactor(bloom, float32(mip), float32(mipCount-1)))

	pass.SetPipeline(pipeline.Get())
	pass.SetBlendConstant(&wgpu.Color{R: bf, G: bf, B: bf, A: 1.0})
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)

	pass.End()
}

func bloomPrepareRenderPass(ctx *RenderContext, enc *wgpu.CommandEncoder, pipeline Pipeline, source, target *wgpu.TextureView, uniforms *ComponentUniforms[bloomUniforms], label string, loadOp wgpu.LoadOp) (*wgpu.RenderPassEncoder, *wgpu.BindGroup) {
	bloomSampler := ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "Bloom Sampler",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeLinear,
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
				View:    target,
				LoadOp:  loadOp,
				StoreOp: wgpu.StoreOpStore,
			},
		},
	})

	return pass, bindGroup
}

func bloomGetTexture(commands *byke.Commands, cache *TextureCache, bv bloomViewQuery) bloomTexture {
	mipCount := uint32(max(2, bits.Len32(bv.Bloom.MaxMipDimension)) - 1)

	cameraSize := bv.ViewTarget.Size

	var mipHeightRatio float32
	if h := cameraSize[1]; h != 0 {
		mipHeightRatio = float32(bv.Bloom.MaxMipDimension) / h
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

	if bv.Texture.Texture == texture {
		// reuse previous bloom texture
		return bv.Texture
	}

	// we can free the old views early
	bv.Texture.Release()

	// and now create new views
	bloomTexture := makeBloomTexture(texture, mipCount)
	commands.Entity(bv.Entity).Insert(bloomTexture)
	return bloomTexture
}

type bloomTexture struct {
	byke.Component[bloomTexture]

	Texture *Texture
	views   []*wgpu.TextureView
}

func makeBloomTexture(texture *Texture, mipCount uint32) bloomTexture {
	bt := bloomTexture{Texture: texture}

	for level := uint32(0); level < mipCount; level++ {
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
