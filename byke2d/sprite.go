package byke2d

import (
	_ "embed"
	"log/slog"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Sprite]()

//go:embed sprite.wgsl
var spritesShader string

type Sprite struct {
	byke.ComparableComponent[Sprite]
	Texture *Texture

	// Sets a custom render size for this texture.
	// The default is to render at the textures native size.
	CustomSize Optional[glm.Vec2f]

	// flips the sprite during rendering.
	FlipX, FlipY bool
}

type renderSpriteValue struct {
	Sprite Sprite
}

type renderSpritePipelineConfig struct {
	Format      wgpu.TextureFormat
	SampleCount uint32
}

func (r renderSpritePipelineConfig) Specialize(dev *wgpu.Device) *wgpu.RenderPipeline {
	module := dev.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "SpriteShaderModule",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: spritesShader},
	})

	return dev.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "SpriteRenderPipeline",
		Vertex: wgpu.VertexState{
			Module:     module,
			EntryPoint: "vs_main",
		},
		Fragment: &wgpu.FragmentState{
			Module:     module,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    r.Format,
					Blend:     &wgpu.BlendStateAlphaBlending,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCW,
			CullMode:  wgpu.CullModeNone,
		},
		Multisample: wgpu.MultisampleState{
			Count:                  r.SampleCount,
			Mask:                   0xffffffff,
			AlphaToCoverageEnabled: false,
		},
	})
}

type PipelineConfig = wx.PipelineConfig

type PipelineCache struct {
	pipelines map[any]any
}

func makePipelineCache() PipelineCache {
	return PipelineCache{
		pipelines: map[any]any{},
	}
}

func pipelineCacheGet[C PipelineConfig](cache *PipelineCache, ctx *RenderContext, config C) wx.CachedPipeline {
	configType := reflect.TypeFor[C]()

	pipelineCache := cache.pipelines[configType]
	if pipelineCache == nil {
		pipelineCache = wx.NewPipelineCache[C](ctx.Context)
		cache.pipelines[configType] = pipelineCache
	}

	return pipelineCache.(*wx.PipelineCache[C]).Get(config)
}

func renderSpriteSystem(
	query byke.Query[renderSpriteValue],
	ctx *RenderContext,
	viewTarget *ViewTarget,
	bufIndices *byke.Local[*wgpu.Buffer],
	pipelineCache *PipelineCache,
) {
	if bufIndices.Value == nil {
		slog.Debug("Initialize index buffer")

		bufIndices.Value = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite.Indices",
			Contents: wgpu.ToBytes([]uint16{2, 0, 1, 1, 3, 2}),
			Usage:    wgpu.BufferUsageIndex,
		})
	}

	conf := renderSpritePipelineConfig{
		Format:      viewTarget.Format,
		SampleCount: 1,
	}

	cp := pipelineCacheGet(pipelineCache, ctx, conf)

	encoder := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Sprite.CommandEncoder"})
	defer encoder.Release()

	for sprite := range query.Items() {
		bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Sprite.BindGroup",
			Layout: cp.GetBindGroupLayout(0),
			Entries: []wgpu.BindGroupEntry{
				{
					Binding:     0,
					TextureView: sprite.Sprite.Texture.TextureView,
				},
				{
					Binding: 1,
					Sampler: sprite.Sprite.Texture.Sampler,
				},
			},
		})

		pass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			Label: "Sprite.RenderPass",
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				{
					View:    viewTarget.Target,
					LoadOp:  wgpu.LoadOpLoad,
					StoreOp: wgpu.StoreOpStore,
				},
			},
		})

		pass.SetPipeline(cp.Pipeline)
		pass.SetBindGroup(0, bindGroup, nil)
		pass.SetIndexBuffer(bufIndices.Value, wgpu.IndexFormatUint16, 0, wgpu.WholeSize)
		pass.DrawIndexed(6, 1, 0, 0, 0)
		pass.End()
		
		bindGroup.Release()
	}

	ctx.Queue.Submit(encoder.Finish(nil))
}
