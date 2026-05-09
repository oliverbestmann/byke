package byke2d

import (
	_ "embed"
	"slices"
	"strconv"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Anchor]()

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
	Color Color

	// flips the sprite during rendering.
	FlipX, FlipY bool
}

func (Sprite) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
		AnchorCenter,
		InheritVisibility,
	}
}

type Anchor struct {
	byke.ComparableComponent[Anchor]
	glm.Vec2f
}

var (
	AnchorTopLeft      = &Anchor{Vec2f: glm.Vec2f{-0.5, -0.5}}
	AnchorTopCenter    = &Anchor{Vec2f: glm.Vec2f{0.0, -0.5}}
	AnchorTopRight     = &Anchor{Vec2f: glm.Vec2f{0.5, -0.5}}
	AnchorCenterLeft   = &Anchor{Vec2f: glm.Vec2f{-0.5, 0}}
	AnchorCenter       = &Anchor{Vec2f: glm.Vec2f{0.0, 0}}
	AnchorCenterRight  = &Anchor{Vec2f: glm.Vec2f{0.5, 0}}
	AnchorBottomLeft   = &Anchor{Vec2f: glm.Vec2f{-0.5, 0.5}}
	AnchorBottomCenter = &Anchor{Vec2f: glm.Vec2f{0.0, 0.5}}
	AnchorBottomRight  = &Anchor{Vec2f: glm.Vec2f{0.5, 0.5}}
)

type renderSpritePipelineConfig struct {
	Format      wgpu.TextureFormat
	SampleCount uint32
}

func (r renderSpritePipelineConfig) Specialize(ctx *RenderContext) *wgpu.RenderPipeline {
	module := ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "SpriteShaderModule",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: spritesShader},
	})

	return ctx.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
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

type renderSpriteValue struct {
	Sprite       Sprite
	Transform    GlobalTransform
	Visibility   ComputedVisibility
	TextureAtlas byke.Option[TextureAtlas]
	RenderLayers byke.Option[RenderLayers]
	Anchor       Anchor
}

type renderSpriteAllocations struct {
	bufIndices        *wgpu.Buffer
	bufView           *wgpu.Buffer
	bufInstances      *wgpu.Buffer
	bufAlphaOnlyTrue  *wgpu.Buffer
	bufAlphaOnlyFalse *wgpu.Buffer

	instances    wx.InstanceWriter
	spritesSlice []renderSpriteValue
}

func renderSpriteSystem(
	camera CurrentCamera,
	spritesQuery byke.Query[renderSpriteValue],
	ctx *RenderContext,
	cachedAllocs *byke.Local[renderSpriteAllocations],
	pipelines Pipelines[renderSpritePipelineConfig],
) {
	defer puffin.NewScope("byke2d.RenderSprites").End()

	const bufInstancesSize = 1024 * 1024
	allocs := &cachedAllocs.Value

	if allocs.bufIndices == nil {
		defer puffin.NewScope("allocate buffers").End()

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

		allocs.bufAlphaOnlyTrue = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite.AlphaOnlyTrueUniform",
			Usage:    wgpu.BufferUsageUniform,
			Contents: wgpu.ToBytes([]uint32{1}),
		})

		allocs.bufAlphaOnlyFalse = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite.AlphaOnlyFalseUniform",
			Usage:    wgpu.BufferUsageUniform,
			Contents: wgpu.ToBytes([]uint32{0}),
		})
	}

	// collect instances values, reuse allocations
	instances := &allocs.instances
	instances.Clear()

	const instanceSize = 52

	conf := renderSpritePipelineConfig{
		Format:      camera.ViewTarget.Format,
		SampleCount: camera.ViewTarget.SampleCount,
	}

	pipeline := pipelines.Specialize(conf)

	vv := ViewValues{
		Transform:   camera.Transform,
		Projection:  camera.Projection,
		SurfaceSize: camera.ViewTarget.Size,
	}

	viewUniformValue := viewUniform{
		ScreenToNDC:   vv.SurfaceToNDC(),
		WorldToScreen: vv.CameraToSurface().Mul(vv.WorldToCamera()),
	}

	// upload uniforms for this camera
	ctx.Queue.WriteBuffer(allocs.bufView, 0, viewUniformValue.ToWGPU())

	// bind ViewUniform
	viewBindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Sprite.ViewUniform.BindGroup",
		Layout: pipeline.GetBindGroupLayout(0),
		Entries: Sequential(
			BindingBuffer(allocs.bufView),
		),
	})

	type batchKey struct {
		Texture *Texture
	}

	flush := func(key batchKey) {
		defer puffin.NewScopeWithValue("Flush", strconv.Itoa(instances.Count())).End()

		encoder := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Sprite.CommandEncoder"})
		defer encoder.Release()

		bytesInstances := instances.Bytes()
		ctx.WriteBuffer(allocs.bufInstances, 0, bytesInstances)

		// interpret one channel textures as alpha only
		var bufAlphaOnly = cachedAllocs.Value.bufAlphaOnlyFalse
		if key.Texture.Descriptor.Format == wgpu.TextureFormatR8Unorm {
			bufAlphaOnly = cachedAllocs.Value.bufAlphaOnlyTrue
		}

		bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Sprite.BindGroup",
			Layout: pipeline.GetBindGroupLayout(1),
			Entries: Sequential(
				BindingTextureView(key.Texture.TextureView),
				BindingSampler(key.Texture.Sampler),
				BindingBuffer(bufAlphaOnly),
			),
		})

		defer bindGroup.Release()

		pass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
			Label: "Sprite.RenderPass",
			ColorAttachments: []wgpu.RenderPassColorAttachment{
				camera.ViewTarget.Attachment(),
			},
		})
		defer pass.Release()

		pass.SetPipeline(pipeline.Get())
		pass.SetBindGroup(0, viewBindGroup, nil)
		pass.SetBindGroup(1, bindGroup, nil)
		pass.SetVertexBuffer(0, allocs.bufInstances, 0, uint64(len(bytesInstances)))
		pass.SetIndexBuffer(allocs.bufIndices, wgpu.IndexFormatUint16, 0, wgpu.WholeSize)
		pass.DrawIndexed(6, uint32(instances.Count()), 0, 0, 0)
		pass.End()

		ctx.Submit(encoder.Finish(nil))
	}

	var keyCurrent batchKey

	sprites := spritesQuery.AppendTo(allocs.spritesSlice[:0])
	allocs.spritesSlice = sprites

	// sort sprites by z-order
	slices.SortFunc(sprites, func(a, b renderSpriteValue) int {
		aZ := a.Transform.Translation[2]
		bZ := b.Transform.Translation[2]
		switch {
		case aZ < bZ:
			return -1
		case aZ > bZ:
			return 1
		default:
			return 0
		}
	})

	for _, sprite := range sprites {
		if !sprite.Visibility.Visible {
			continue
		}

		if !camera.RenderLayers.Intersects(sprite.RenderLayers.Or(renderLayerZero)) {
			continue
		}

		key := batchKey{
			Texture: sprite.Sprite.Texture,
		}

		hasNoSpace := bufInstancesSize-len(instances.Bytes()) < instanceSize
		hasNewTexture := instances.Count() > 0 && keyCurrent != key

		if instances.Count() > 0 && (hasNewTexture || hasNoSpace) {
			flush(keyCurrent)
			instances.Clear()
		}

		keyCurrent = key

		// display the full image by default
		textureSize := sprite.Sprite.Texture.Size()
		rect := glm.RectFromPoints(glm.Vec2f{}, textureSize)

		// but apply texture atlas if available
		if ta, ok := sprite.TextureAtlas.Get(); ok {
			if current, ok := ta.Current(); ok {
				rect.Min = current.Min.ToVec2f()
				rect.Max = current.Max.ToVec2f()
			}
		}

		// uv = offset + position * scale
		uvOffset := rect.Min.Div(textureSize)
		uvScale := rect.Size().Div(textureSize)

		if sprite.Sprite.FlipX {
			// flip uv along the x axis
			uvOffset[0] += uvScale[0]
			uvScale[0] *= -1
		}

		if !sprite.Sprite.FlipY {
			// flip uv along the y axis
			uvOffset[1] += uvScale[1]
			uvScale[1] *= -1
		}

		// calculate size of the sprite
		baseSize := sprite.Sprite.CustomSize.Or(rect.Size())
		scale := sprite.Transform.Scale.Truncate().Mul(baseSize)
		anchorOffset := sprite.Anchor.Mul(glm.Vec2f{-1, 1}).Add(glm.Vec2f{-0.5, -0.5}).Mul(scale)

		instances.StartNew(instanceSize)

		// @location(0) i_translation: vec2<f32>,
		instances.AppendVec2f(sprite.Transform.Translation.Truncate().Add(anchorOffset))
		// @location(1) i_scale: vec2<f32>,
		instances.AppendVec2f(scale)
		// @location(2) i_rotation: f32,
		instances.AppendFloat32(float32(sprite.Transform.Rotation))
		// @location(3) i_uv_offset: vec2<f32>,
		instances.AppendVec2f(uvOffset)
		// @location(4) i_uv_scale: vec2<f32>,
		instances.AppendVec2f(uvScale)
		// @location(5) i_color: vec4<f32>,
		instances.AppendVec4f(sprite.Sprite.Color.ToVec())
	}

	if instances.Count() > 0 {
		flush(keyCurrent)
		instances.Clear()
	}

	viewBindGroup.Release()
}
