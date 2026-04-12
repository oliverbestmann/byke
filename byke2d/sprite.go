package byke2d

import (
	_ "embed"
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

	// The texture to use
	Texture *Texture

	// Sets a custom render size for this texture.
	// The default is to render at the textures native size.
	CustomSize Optional[glm.Vec2f]

	// A color tint for this sprite.
	Color wx.Color

	// flips the sprite during rendering.
	FlipX, FlipY bool
}

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return append(
		[]byke.ErasedComponent{NewTransform(), AnchorCenter, InheritVisibility},
	)
}

type Anchor struct {
	byke.ComparableComponent[Anchor]
	glm.Vec2f
}

var (
	AnchorTopLeft      = &Anchor{Vec2f: glm.Vec2f{0.0, 0.0}}
	AnchorTopCenter    = &Anchor{Vec2f: glm.Vec2f{0.5, 0.0}}
	AnchorTopRight     = &Anchor{Vec2f: glm.Vec2f{1.0, 0.0}}
	AnchorCenterLeft   = &Anchor{Vec2f: glm.Vec2f{0, 0.5}}
	AnchorCenter       = &Anchor{Vec2f: glm.Vec2f{0.5, 0.5}}
	AnchorCenterRight  = &Anchor{Vec2f: glm.Vec2f{1.0, 0.5}}
	AnchorBottomLeft   = &Anchor{Vec2f: glm.Vec2f{0, 1.0}}
	AnchorBottomCenter = &Anchor{Vec2f: glm.Vec2f{0.5, 1.0}}
	AnchorBottomRight  = &Anchor{Vec2f: glm.Vec2f{1.0, 1.0}}
)

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
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 52,
					StepMode:    wgpu.VertexStepModeInstance,
					Attributes: []wgpu.VertexAttribute{
						// @location(0) i_translation: vec2<f32>,
						// @location(1) i_scale: vec2<f32>,
						// @location(2) i_rotation: f32,
						// @location(3) i_uv_offset: vec2<f32>,
						// @location(4) i_uv_scale: vec2<f32>,
						// @location(5) i_color: vec4<f32>,

						{
							ShaderLocation: 0,
							Offset:         0,
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 1,
							Offset:         8,
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 2,
							Offset:         16,
							Format:         wgpu.VertexFormatFloat32,
						},
						{
							ShaderLocation: 3,
							Offset:         20,
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 4,
							Offset:         28,
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 5,
							Offset:         36,
							Format:         wgpu.VertexFormatFloat32x4,
						},
					},
				},
			},
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

type renderCameraValue struct {
	Camera     Camera
	Projection OrthographicProjection
	Transform  GlobalTransform
}

type renderSpriteValue struct {
	Sprite       Sprite
	Transform    GlobalTransform
	Visibility   ComputedVisibility
	TextureAtlas byke.Option[TextureAtlas]
}

type renderSpriteAllocations struct {
	bufIndices   *wgpu.Buffer
	bufView      *wgpu.Buffer
	bufInstances *wgpu.Buffer
}

func renderSpriteSystem(
	spritesQuery byke.Query[renderSpriteValue],
	ctx *RenderContext,
	viewTarget *ViewTarget,
	cachedAllocs *byke.Local[renderSpriteAllocations],
	pipelineCache *PipelineCache,
	camerasQuery byke.Query[renderCameraValue],
) {
	const bufInstancesSize = 1024 * 1024
	allocs := &cachedAllocs.Value

	if allocs.bufIndices == nil {
		allocs.bufIndices = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite.Indices",
			Contents: wgpu.ToBytes([]uint16{2, 0, 1, 1, 3, 2}),
			Usage:    wgpu.BufferUsageIndex,
		})

		allocs.bufView = ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "Sprite.ViewUniform",
			Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
			Size:  max(96, uint64(len(new(viewUniform).ToWGPU()))),
		})

		allocs.bufInstances = ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "Sprite.Instances",
			Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
			Size:  bufInstancesSize,
		})
	}

	conf := renderSpritePipelineConfig{
		Format:      viewTarget.Format,
		SampleCount: viewTarget.SampleCount,
	}

	cp := pipelineCacheGet(pipelineCache, ctx, conf)

	// collect instances values, reuse allocations
	var instances wx.InstanceWriter
	var instanceCount uint32

	const instanceSize = 52

	for camera := range camerasQuery.Items() {
		screenSize := camera.Projection.ScalingMode.ViewportSize(viewTarget.Size.XY())

		viewUniformValue := viewUniform{
			ScreenToNDC:   camera.Projection.ScreenToNDC(screenSize),
			WorldToScreen: camera.Transform.AsMat3f(),
		}

		// upload uniforms for this camera
		ctx.Queue.WriteBuffer(allocs.bufView, 0, viewUniformValue.ToWGPU())

		// bind ViewUniform
		viewBindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Sprite.ViewUniform.BindGroup",
			Layout: cp.GetBindGroupLayout(0),
			Entries: []wgpu.BindGroupEntry{
				{
					Binding: 0,
					Buffer:  allocs.bufView,
					Size:    wgpu.WholeSize,
				},
			},
		})

		flush := func(currentTexture *Texture) {
			encoder := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Sprite.CommandEncoder"})
			defer encoder.Release()

			bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:  "Sprite.BindGroup",
				Layout: cp.GetBindGroupLayout(1),
				Entries: []wgpu.BindGroupEntry{
					{
						Binding:     0,
						TextureView: currentTexture.TextureView,
					},
					{
						Binding: 1,
						Sampler: currentTexture.Sampler,
					},
				},
			})

			defer bindGroup.Release()

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
			defer pass.Release()

			ctx.WriteBuffer(allocs.bufInstances, 0, instances.Bytes())

			pass.SetPipeline(cp.Pipeline)
			pass.SetBindGroup(0, viewBindGroup, nil)
			pass.SetBindGroup(1, bindGroup, nil)
			pass.SetVertexBuffer(0, allocs.bufInstances, 0, uint64(instanceCount)*instanceSize)
			pass.SetIndexBuffer(allocs.bufIndices, wgpu.IndexFormatUint16, 0, wgpu.WholeSize)
			pass.DrawIndexed(6, instanceCount, 0, 0, 0)
			pass.End()

			ctx.Submit(encoder.Finish(nil))
		}

		var currentTexture *Texture = nil

		for sprite := range spritesQuery.Items() {
			hasNoSpace := bufInstancesSize-instances.Len() < instanceSize
			hasNewTexture := currentTexture != nil && currentTexture != sprite.Sprite.Texture

			if hasNewTexture || hasNoSpace {
				flush(currentTexture)
				instances.Clear()
				instanceCount = 0
			}

			// display the full image by default
			textureSize := sprite.Sprite.Texture.Size()
			rect := wx.RectangleFromPoints(glm.Vec2f{}, textureSize)

			// but apply texture atlas if available
			if ta, ok := sprite.TextureAtlas.Get(); ok {
				idx := ta.Index % len(ta.Layout)
				rect.Min = ta.Layout[idx].Min.ToVec2f()
				rect.Max = ta.Layout[idx].Max.ToVec2f()
			}

			currentTexture = sprite.Sprite.Texture

			// initial base size of the sprite
			baseSize := sprite.Sprite.CustomSize.Or(rect.Size())

			if sprite.Sprite.FlipX {
				baseSize[0] *= -1
			}

			if sprite.Sprite.FlipY {
				baseSize[1] *= -1
			}

			uvOffset := rect.Min.Div(textureSize)
			uvScale := rect.Size().Div(textureSize)

			// @location(0) i_translation: vec2<f32>,
			instances.AppendVec2f(sprite.Transform.Translation.Truncate())
			// @location(1) i_scale: vec2<f32>,
			instances.AppendVec2f(sprite.Transform.Scale.Truncate().Mul(baseSize))
			// @location(2) i_rotation: f32,
			instances.AppendFloat32(float32(sprite.Transform.Rotation))
			// @location(3) i_uv_offset: vec2<f32>,
			instances.AppendVec2f(uvOffset)
			// @location(4) i_uv_scale: vec2<f32>,
			instances.AppendVec2f(uvScale)
			// @location(5) i_color: vec4<f32>,
			instances.AppendVec4f(sprite.Sprite.Color.ToVec())

			instanceCount += 1
		}

		if instanceCount > 0 {
			flush(currentTexture)
			instances.Clear()
			instanceCount = 0
		}

		viewBindGroup.Release()
	}
}
