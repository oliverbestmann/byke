package byke2d

import (
	_ "embed"
	"sort"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Anchor]()

var _ = byke.ValidateComponent[bindGroupsSprites]()
var _ = byke.ValidateComponent[metaSprites]()

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

func pluginSprite(app *byke.App) {
	app.InsertResource(staticSpriteUniforms{})
	app.InsertResource(ExtractedSprites{})

	app.AddSystems(Render,
		byke.System(extractSpritesSystem).InSet(RenderPhaseExtract),
		byke.System(uploadSpritesSystem).InSet(RenderPhasePrepareResources),
		byke.System(prepareBindGroupsSpritesSystem).InSet(RenderPhasePrepareBindGroups),
		byke.System(clearExtractedSpritesSystem).InSet(RenderPhaseCleanup),
	)

	app.AddSystems(Core2d,
		byke.
			System(renderSpritesSystem).
			InSet(Core2dMain))
}

type renderSpritePipelineConfig struct {
	Shader      *ShaderDef
	Format      wgpu.TextureFormat
	SampleCount uint32
}

func (r renderSpritePipelineConfig) Specialize(ctx PipelineContext) RenderPipelineDescriptor {
	var shaderLabel = "Sprite"
	var shaderSource = "#import byke2d::sprite"
	var entryVertex = "vs_main"
	var entryFragment = "fs_main"
	var shaderValues ShaderValues

	if r.Shader != nil {
		shaderLabel = valueOr(r.Shader.Label, "Custom Sprite Shader")
		shaderSource = r.Shader.Source
		shaderValues = r.Shader.Values
		entryVertex = valueOr(r.Shader.VertexEntry, entryVertex)
		entryFragment = valueOr(r.Shader.FragmentEntry, entryFragment)
	}

	var module = ctx.Shader(shaderLabel, shaderSource, shaderValues)

	return RenderPipelineDescriptor{
		Label: "SpriteRenderPipeline",
		Layout: []wgpu.BindGroupLayoutDescriptor{
			layoutView,
			layoutTextures,
		},
		Vertex: wgpu.VertexState{
			Module:     module,
			EntryPoint: entryVertex,
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 60,
					StepMode:    wgpu.VertexStepModeInstance,
					Attributes: []wgpu.VertexAttribute{
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
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 3,
							Offset:         24,
							Format:         wgpu.VertexFormatFloat32,
						},
						{
							ShaderLocation: 4,
							Offset:         28,
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 5,
							Offset:         36,
							Format:         wgpu.VertexFormatFloat32x2,
						},
						{
							ShaderLocation: 6,
							Offset:         44,
							Format:         wgpu.VertexFormatFloat32x4,
						},
					},
				},
			},
		},
		Fragment: &wgpu.FragmentState{
			Module:     module,
			EntryPoint: entryFragment,
			Targets: []wgpu.ColorTargetState{
				{
					Format:    r.Format,
					Blend:     &wgpu.BlendStateAlphaBlending,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},

		Multisample: multisampleState(r.SampleCount),
	}
}

func valueOr[T comparable](first, fallback T) T {
	var zero T
	if first != zero {
		return first
	}

	return fallback
}

// ExtractedSprite got extracted from the World in Prepare and will be rendered
// to the screen. Does not need to be backed by a real sprite.
type ExtractedSprite struct {
	Texture      *Texture
	Transform    GlobalTransform
	Color        Color
	Rect         glm.Rect2f
	Size         glm.Vec2f
	Anchor       Anchor
	RenderLayers RenderLayers
	FlipX, FlipY bool

	// optional custom shader definition to replace or extend the
	// sprites default shader.
	CustomShader *ShaderDef
}

type ExtractedSprites struct {
	Sprites []ExtractedSprite
}

func clearExtractedSpritesSystem(
	sprites *ExtractedSprites,
) {
	sprites.Sprites = sprites.Sprites[:0]
}

// extractSpritesSystem adds the ExtractedSprite component to all renderable
// entities that have a Sprite component.
func extractSpritesSystem(
	sprites *ExtractedSprites,
	spritesQuery byke.Query[struct {
		byke.EntityId
		Sprite       Sprite
		Transform    GlobalTransform
		Visibility   ComputedVisibility
		TextureAtlas byke.Option[TextureAtlas]
		RenderLayers byke.Option[RenderLayers]
		CustomShader byke.Option[CustomShader]
		Anchor       Anchor
	}],
) {
	for sprite := range spritesQuery.Items() {
		if !sprite.Visibility.Visible {
			continue
		}

		// calculate size of the rect to display
		rect := glm.RectFromSize(glm.Vec2f{}, sprite.Sprite.Texture.Size())

		// but apply texture atlas if available
		if ta, ok := sprite.TextureAtlas.Get(); ok {
			if current, ok := ta.Current(); ok {
				rect.Min = current.Min.ToVec2f()
				rect.Max = current.Max.ToVec2f()
			}
		}

		sprites.Sprites = append(sprites.Sprites, ExtractedSprite{
			Texture:      sprite.Sprite.Texture,
			Color:        sprite.Sprite.Color,
			Size:         sprite.Sprite.CustomSize.Or(rect.Size()),
			FlipX:        sprite.Sprite.FlipX,
			FlipY:        sprite.Sprite.FlipY,
			RenderLayers: sprite.RenderLayers.Or(renderLayerAll),
			Transform:    sprite.Transform,
			Anchor:       sprite.Anchor,
			Rect:         rect,
			CustomShader: sprite.CustomShader.OrZero().Shader,
		})
	}
}

type metaSprites struct {
	byke.Component[metaSprites]
	Instances  wx.InstanceWriter
	Buffer     *wgpu.Buffer
	BufferSize int
	Batches    []spritesBatch
}

func (m metaSprites) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		bindGroupsSprites{},
	}
}

type spritesBatch struct {
	Shader        *ShaderDef
	Texture       *Texture
	Offset        uint64
	Size          uint64
	InstanceCount uint32
}

func uploadSpritesSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		EntityId     byke.EntityId
		Meta         byke.OptionMut[metaSprites]
		RenderLayers RenderLayers
	}],
	sprites *ExtractedSprites,
) {
	// sort sprites by z-order
	sort.Slice(sprites.Sprites, func(a, b int) bool {
		aZ := sprites.Sprites[a].Transform.Translation[2]
		bZ := sprites.Sprites[b].Transform.Translation[2]
		return aZ < bZ
	})

	for view := range viewsQuery.Items() {
		meta, metaSet := view.Meta.Get()
		if !metaSet {
			meta = &metaSprites{}
		}

		// the size of one sprite instance in the wgpu instance buffer
		const instanceSize = 60

		instances := &meta.Instances

		meta.Batches = meta.Batches[:0]
		meta.Instances.Clear()

		var batchStart uint64
		var batchTexture *Texture
		var batchShader *ShaderDef

		maybeFlush := func(nextTexture *Texture, nextShader *ShaderDef) {
			if batchTexture == nextTexture && batchShader == nextShader {
				return
			}

			byteCount := meta.Instances.ByteCount()

			if batchStart != byteCount && batchTexture != nil {
				// start a new batch, flush the previous one
				meta.Batches = append(meta.Batches, spritesBatch{
					Shader:        batchShader,
					Texture:       batchTexture,
					Offset:        batchStart,
					Size:          byteCount - batchStart,
					InstanceCount: uint32((byteCount - batchStart) / instanceSize),
				})
			}

			batchStart = byteCount
			batchTexture = nextTexture
			batchShader = nextShader
		}

		for _, sp := range sprites.Sprites {
			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				// not rendered by this camera
				continue
			}

			maybeFlush(sp.Texture, sp.CustomShader)

			textureSize := sp.Texture.Size()

			// uv = offset + position * scale
			uvOffset := sp.Rect.Min.Div(textureSize)
			uvScale := sp.Rect.Size().Div(textureSize)

			if sp.FlipX {
				// flip uv along the x axis
				uvOffset[0] += uvScale[0]
				uvScale[0] *= -1
			}

			if !sp.FlipY {
				// flip uv along the y axis
				uvOffset[1] += uvScale[1]
				uvScale[1] *= -1
			}

			// calculate size of the sprite
			baseSize := sp.Size
			scale := sp.Transform.Scale.Truncate().Mul(baseSize)
			// anchorOffset := sp.Anchor.Mul(glm.Vec2f{-1, 1}).Add(glm.Vec2f{-0.5, -0.5}).Mul(scale)

			instances.StartNew(instanceSize)

			// @location(0) i_translation: vec2<f32>,
			instances.AppendVec2f(sp.Transform.Translation.Truncate())
			// @location(1) i_scale: vec2<f32>,
			instances.AppendVec2f(scale)
			// @location(2) i_anchor: vec2<f32>,
			instances.AppendVec2f(sp.Anchor.Vec2f)
			// @location(3) i_rotation: f32,
			instances.AppendFloat32(float32(sp.Transform.Rotation))
			// @location(4) i_uv_offset: vec2<f32>,
			instances.AppendVec2f(uvOffset)
			// @location(5) i_uv_scale: vec2<f32>,
			instances.AppendVec2f(uvScale)
			// @location(6) i_color: vec4<f32>,
			instances.AppendVec4f(sp.Color.ToVec())
		}

		// flush the last batch if needed
		maybeFlush(nil, nil)

		data := instances.Bytes()

		if meta.BufferSize < len(data) && meta.Buffer != nil {
			meta.Buffer.Release()
			meta.Buffer = nil
			meta.BufferSize = 0
		}

		if meta.Buffer == nil {
			meta.BufferSize = max(4096, len(data))

			meta.Buffer = ctx.CreateBuffer(&wgpu.BufferDescriptor{
				Label: "Sprite Instances",
				Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageVertex,
				Size:  uint64(meta.BufferSize),
			})
		}

		// upload data to buffer
		ctx.WriteBuffer(meta.Buffer, 0, data)

		if !metaSet {
			commands.Entity(view.EntityId).Insert(meta)
		}
	}
}

type bindGroupsSprites struct {
	byke.Component[bindGroupsSprites]
	View    *wgpu.BindGroup
	Batches []*wgpu.BindGroup
}

func (c *bindGroupsSprites) Reset() {
	if c.View != nil {
		// TODO somehow reuse if not changed
		c.View.Release()
	}

	for _, batch := range c.Batches {
		batch.Release()
	}

	// clear the pointers
	clear(c.Batches)

	c.Batches = c.Batches[:0]
}

type staticSpriteUniforms struct {
	alphaOnlyFalse *wgpu.Buffer
	alphaOnlyTrue  *wgpu.Buffer
}

func (s *staticSpriteUniforms) AlphaOnly(ctx *RenderContext, value bool) wgpu.BindGroupEntry {
	if s.alphaOnlyTrue == nil {
		s.alphaOnlyTrue = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite Uniform AlphaOnlyTrue",
			Usage:    wgpu.BufferUsageUniform,
			Contents: wgpu.ToBytes([]uint32{1}),
		})

		s.alphaOnlyFalse = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Sprite Uniform AlphaOnlyFalse",
			Usage:    wgpu.BufferUsageUniform,
			Contents: wgpu.ToBytes([]uint32{0}),
		})
	}

	if value {
		return BindingBuffer(s.alphaOnlyTrue)
	}

	return BindingBuffer(s.alphaOnlyFalse)
}

var layoutView = SequentialLayoutWithLabel("ViewUniforms",
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
)

var layoutTextures = SequentialLayoutWithLabel("Spite Textures",
	BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
	BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false),
)

func prepareBindGroupsSpritesSystem(
	ctx *RenderContext,

	views byke.Query[struct {
		ViewTarget *ViewTarget
		Meta       *metaSprites
		BindGroups *bindGroupsSprites
	}],

	pipelines *PipelineCache,
	viewUniforms *ComponentUniforms[ViewUniforms],
	staticSpriteUniforms *staticSpriteUniforms,
) {
	for view := range views.Items() {
		view.BindGroups.View = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Sprite.ViewUniform.BindGroup",
			Layout: pipelines.BindGroupLayout(layoutView),
			Entries: Sequential(
				viewUniforms.Binding(),
			),
		})

		for _, batch := range view.Meta.Batches {
			alphaOnly := batch.Texture.Descriptor.Format == wgpu.TextureFormatR8Unorm
			bufAlphaOnly := staticSpriteUniforms.AlphaOnly(ctx, alphaOnly)

			bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:  "Sprite.BindGroup",
				Layout: pipelines.BindGroupLayout(layoutTextures),
				Entries: Sequential(
					BindingTextureView(batch.Texture.TextureView),
					BindingSampler(batch.Texture.Sampler),
					bufAlphaOnly,
				),
			})

			view.BindGroups.Batches = append(view.BindGroups.Batches, bindGroup)
		}
	}
}

func renderSpritesSystem(
	ctx *RenderContext,
	pipelines Pipelines[renderSpritePipelineConfig],
	viewQuery ViewQuery[struct {
		Camera             *Camera
		ViewTarget         *ViewTarget
		Meta               *metaSprites
		BindGroups         *bindGroupsSprites
		ViewUniformsOffset DynamicOffset[ViewUniforms]
	}],
) {
	view := viewQuery.Get()

	if len(view.Meta.Batches) == 0 {
		return
	}

	enc := ctx.CreateCommandEncoder(nil)
	defer enc.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Sprites",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			view.ViewTarget.Attachment(),
		},
	})

	// bind the view uniforms
	pass.SetBindGroup(0, view.BindGroups.View, []uint32{view.ViewUniformsOffset.Offset})

	for idx, batch := range view.Meta.Batches {
		pipeline := pipelines.Specialize(renderSpritePipelineConfig{
			Shader:      batch.Shader,
			Format:      view.ViewTarget.Format,
			SampleCount: view.ViewTarget.SampleCount,
		})

		pass.SetPipeline(pipeline.Get())
		pass.SetBindGroup(1, view.BindGroups.Batches[idx], nil)
		pass.SetVertexBuffer(0, view.Meta.Buffer, batch.Offset, batch.Size)
		pass.Draw(6, batch.InstanceCount, 0, 0)
	}

	pass.End()

	ctx.Submit(enc.Finish(nil))
}
