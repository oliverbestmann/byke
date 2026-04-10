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

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return append(
		[]byke.ErasedComponent{NewTransform()},
	)
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
	Sprite    Sprite
	Transform GlobalTransform
}

type renderSpriteAllocations struct {
	bufIndices *wgpu.Buffer
	bufView    *wgpu.Buffer
}

func renderSpriteSystem(
	spritesQuery byke.Query[renderSpriteValue],
	ctx *RenderContext,
	viewTarget *ViewTarget,
	gpu *byke.Local[renderSpriteAllocations],
	pipelineCache *PipelineCache,
	camerasQuery byke.Query[renderCameraValue],
) {
	if gpu.Value.bufIndices == nil {
		slog.Debug("Initialize index buffer")

		gpu.Value.bufIndices = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite.Indices",
			Contents: wgpu.ToBytes([]uint16{2, 0, 1, 1, 3, 2}),
			Usage:    wgpu.BufferUsageIndex,
		})
	}

	if gpu.Value.bufView == nil {
		slog.Debug("Initialize view uniform buffer")

		gpu.Value.bufView = ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "Sprite.ViewUniform",
			Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
			Size:  max(96, uint64(len(new(viewUniform).ToWGPU()))),
		})
	}

	conf := renderSpritePipelineConfig{
		Format:      viewTarget.Format,
		SampleCount: 1,
	}

	cp := pipelineCacheGet(pipelineCache, ctx, conf)

	// collect instances values, reuse allocations
	var instances wx.InstanceWriter
	var instanceCount uint32

	for camera := range camerasQuery.Items() {
		screenSize := camera.Projection.ScalingMode.ViewportSize(viewTarget.Size.XY())

		viewUniformValue := viewUniform{
			ScreenToNDC:   camera.Projection.ScreenToNDC(screenSize),
			WorldToScreen: camera.Transform.AsMat3f(),
		}

		// upload uniforms for this camera
		ctx.Queue.WriteBuffer(gpu.Value.bufView, 0, viewUniformValue.ToWGPU())

		viewBindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Sprite.ViewUniform.BindGroup",
			Layout: cp.GetBindGroupLayout(0),
			Entries: []wgpu.BindGroupEntry{
				{
					Binding: 0,
					Buffer:  gpu.Value.bufView,
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

			bufInstances := ctx.CreateBuffer(&wgpu.BufferDescriptor{
				Label: "Sprite.Instances",
				Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
				Size:  uint64(len(instances.Bytes())),
			})

			ctx.WriteBuffer(bufInstances, 0, instances.Bytes())

			pass.SetPipeline(cp.Pipeline)
			pass.SetBindGroup(0, viewBindGroup, nil)
			pass.SetBindGroup(1, bindGroup, nil)
			pass.SetVertexBuffer(0, bufInstances, 0, uint64(instanceCount)*52)
			pass.SetIndexBuffer(gpu.Value.bufIndices, wgpu.IndexFormatUint16, 0, wgpu.WholeSize)
			pass.DrawIndexed(6, instanceCount, 0, 0, 0)
			pass.End()

			ctx.Submit(encoder.Finish(nil))
		}

		var currentTexture *Texture = nil

		for sprite := range spritesQuery.Items() {
			if currentTexture != nil && currentTexture != sprite.Sprite.Texture {
				flush(currentTexture)
				instances.Clear()
				instanceCount = 0
			}

			currentTexture = sprite.Sprite.Texture

			// @location(0) i_translation: vec2<f32>,
			// @location(1) i_scale: vec2<f32>,
			// @location(2) i_rotation: f32,
			// @location(3) i_uv_offset: vec2<f32>,
			// @location(4) i_uv_scale: vec2<f32>,
			// @location(5) i_color: vec4<f32>,
			instances.AppendVec2f(sprite.Transform.Translation.Truncate())
			instances.AppendVec2f(sprite.Transform.Scale.Truncate())
			instances.AppendFloat32(float32(sprite.Transform.Rotation))
			instances.AppendVec2f(glm.Vec2f{1, 1})
			instances.AppendVec2f(glm.Vec2f{0, 0})
			instances.AppendVec4f(glm.Vec4f{1, 1, 1, 1})

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
