package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/radix"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/puffin-go"
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

type offsetCalc struct {
	index  uint32
	offset uint64
}

func (o *offsetCalc) Inc(size uint64, fmt wgpu.VertexFormat) wgpu.VertexAttribute {
	attr := wgpu.VertexAttribute{
		ShaderLocation: o.index,
		Offset:         o.offset,
		Format:         fmt,
	}

	o.index += 1
	o.offset += size

	return attr
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

	var offset offsetCalc

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
					ArrayStride: 100,
					StepMode:    wgpu.VertexStepModeInstance,
					Attributes: []wgpu.VertexAttribute{
						offset.Inc(16, wgpu.VertexFormatFloat32x4),
						offset.Inc(16, wgpu.VertexFormatFloat32x4),
						offset.Inc(16, wgpu.VertexFormatFloat32x4),
						offset.Inc(16, wgpu.VertexFormatFloat32x4),
						offset.Inc(8, wgpu.VertexFormatFloat32x2),
						offset.Inc(8, wgpu.VertexFormatFloat32x2),
						offset.Inc(16, wgpu.VertexFormatFloat32x4),
						offset.Inc(4, wgpu.VertexFormatUint32),
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
	Texture *Texture

	// optional custom shader definition to replace or extend the
	// sprites default shader.
	CustomShader *ShaderDef

	Transform    glm.Mat4f
	Color        Color
	Rect         glm.Rectf
	Size         glm.Vec2f
	Anchor       Anchor
	RenderLayers RenderLayers
	ZSort        float32
	FlipX, FlipY bool
}

type ExtractedSprites struct {
	Sprites []ExtractedSprite

	sortCache radix.Cache
	indices   []radix.Value
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
		rect := glm.Rectf{Max: sprite.Sprite.Texture.Size()}

		// but apply texture atlas if available
		if ta, ok := sprite.TextureAtlas.Get(); ok {
			if current, ok := ta.Current(); ok {
				rect.Min = current.Min.ToVec2f()
				rect.Max = current.Max.ToVec2f()
			}
		}

		sprites.Sprites = append(sprites.Sprites, ExtractedSprite{
			Texture:      sprite.Sprite.Texture,
			CustomShader: sprite.CustomShader.OrZero().Shader,
			Color:        sprite.Sprite.Color,
			Size:         sprite.Sprite.CustomSize.Or(rect.Size()),
			FlipX:        sprite.Sprite.FlipX,
			FlipY:        sprite.Sprite.FlipY,
			RenderLayers: sprite.RenderLayers.Or(renderLayerAll),
			Transform:    sprite.Transform.Affine,
			Anchor:       sprite.Anchor,
			Rect:         rect,
			ZSort:        sprite.Transform.Affine.TranslateZ(),
		})
	}
}

type metaSprites struct {
	byke.Component[metaSprites]
	Instances  wgsl.InstanceWriter
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
	sortSprites(sprites)

	for view := range viewsQuery.Items() {
		meta, metaSet := view.Meta.Get()
		if !metaSet {
			meta = &metaSprites{}
		}

		// the size of one sprite instance in the wgpu instance buffer
		const instanceSize = 100

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

		for _, v := range sprites.indices {
			sp := &sprites.Sprites[v.Index]

			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				// not rendered by this camera
				continue
			}

			maybeFlush(sp.Texture, sp.CustomShader)

			textureSize := sp.Texture.Size()

			// uv = offset + position * scale
			uvOffset := sp.Rect.Min.Div(textureSize)

			// TODO: profile again Rect.Size() seems to be slower
			uvScale := sp.Rect.Max.Sub(sp.Rect.Min).Div(textureSize)

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
			transform := sp.Transform.
				Scale(sp.Size.Extend(1.0).XYZ()).
				Translate(-1*sp.Anchor.Vec2f[0]-0.5, sp.Anchor.Vec2f[1]-0.5, 0)

			var flags uint32
			if sp.Texture.Descriptor.Format == wgpu.TextureFormatR8Unorm {
				flags |= 1
			}

			columns := transform.Components()

			instances.StartNew(instanceSize)

			// @location(0) i_affine_0: mat3<f32>,
			instances.AppendVec4f(columns[0])
			// @location(1) i_affine_1: mat3<f32>,
			instances.AppendVec4f(columns[1])
			// @location(2) i_affine_2: mat3<f32>,
			instances.AppendVec4f(columns[2])
			// @location(3) i_affine_3: mat3<f32>,
			instances.AppendVec4f(columns[3])
			// @location(4) i_uv_offset: vec2<f32>,
			instances.AppendVec2f(uvOffset)
			// @location(5) i_uv_scale: vec2<f32>,
			instances.AppendVec2f(uvScale)
			// @location(6) i_color: vec4<f32>,
			instances.AppendVec4f(sp.Color.ToVec())
			// @location(7) i_flags: u32,
			instances.AppendUint(flags)
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

func sortSprites(sprites *ExtractedSprites) {
	defer puffin.NewScope("Sort Sprites").End()

	n := len(sprites.Sprites)
	if n == 0 {
		sprites.indices = sprites.indices[:0]
		return
	}

	if cap(sprites.indices) < n {
		// not enough space, need to allocate
		sprites.indices = make([]radix.Value, n)
	} else {
		// enough space, we can re-use
		sprites.indices = sprites.indices[:n]
	}

	_ = sprites.indices[n-1]

	for idx := range n {
		sprites.indices[idx].Key = sprites.Sprites[idx].ZSort
		sprites.indices[idx].Index = uint32(idx)
	}

	radix.Sort(&sprites.sortCache, sprites.indices)
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

var layoutView = SequentialLayoutWithLabel("ViewUniforms",
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
)

var layoutTextures = SequentialLayoutWithLabel("Spite Textures",
	BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
	BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
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
			bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:  "Sprite.BindGroup",
				Layout: pipelines.BindGroupLayout(layoutTextures),
				Entries: Sequential(
					BindingTextureView(batch.Texture.TextureView),
					BindingSampler(batch.Texture.Sampler),
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
