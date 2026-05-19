package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/radix"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Anchor]()

var _ = byke.ValidateComponent[bindGroupsSprites]()

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

	app.InsertResource(spriteTextureBindGroupCache{})
	app.InsertResource(metaSprites{})

	app.AddSystems(Render,
		byke.System(extractSpritesSystem).InSet(RenderPhaseExtract),
		byke.System(queueSpritesSystem).InSet(RenderPhaseQueue),
		byke.System(prepareSpriteBindGroupsSystem).InSet(RenderPhasePrepareBindGroups),
		byke.System(clearExtractedSpritesSystem).InSet(RenderPhaseCleanup),
	)
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
			SequentialLayoutWithLabel("ViewUniforms",
				BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
			),
			layoutSpriteTextures,
		},
		Vertex: wgpu.VertexState{
			Module:     module,
			EntryPoint: entryVertex,
			Buffers: []wgpu.VertexBufferLayout{
				{
					ArrayStride: 84,
					StepMode:    wgpu.VertexStepModeInstance,
					Attributes: []wgpu.VertexAttribute{
						offset.Inc(12, wgpu.VertexFormatFloat32x3),
						offset.Inc(12, wgpu.VertexFormatFloat32x3),
						offset.Inc(12, wgpu.VertexFormatFloat32x3),
						offset.Inc(12, wgpu.VertexFormatFloat32x3),
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
		Sprite       query.Ref[Sprite]
		Transform    query.Ref[GlobalTransform]
		TextureAtlas byke.Option[TextureAtlas]
		RenderLayers byke.Option[RenderLayers]
		CustomShader byke.Option[CustomShader]
		Anchor       Anchor
		Visibility   ComputedVisibility
	}],
) {
	for item := range spritesQuery.Items() {
		if !item.Visibility.Visible {
			continue
		}

		// calculate size of the rect to display
		sprite := item.Sprite.Value
		rect := glm.Rectf{Max: sprite.Texture.Size()}

		// but apply texture atlas if available
		if ta, ok := item.TextureAtlas.Get(); ok {
			if current, ok := ta.Current(); ok {
				rect.Min = current.Min.ToVec2f()
				rect.Max = current.Max.ToVec2f()
			}
		}

		sprites.Sprites = append(sprites.Sprites, ExtractedSprite{
			Texture:      sprite.Texture,
			CustomShader: item.CustomShader.OrZero().Shader,
			Color:        sprite.Color,
			Size:         sprite.CustomSize.Or(rect.Size()),
			FlipX:        sprite.FlipX,
			FlipY:        sprite.FlipY,
			RenderLayers: item.RenderLayers.Or(renderLayerAll),
			Transform:    item.Transform.Value.Affine,
			Anchor:       item.Anchor,
			Rect:         rect,
		})
	}
}

func queueSpritesSystem(
	sprites *ExtractedSprites,
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		RenderLayers RenderLayers
		RenderPhase  *RenderPhase
	}],
) {
	for view := range viewsQuery.Items() {
		for idx := range sprites.Sprites {
			sp := &sprites.Sprites[idx]
			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				continue
			}

			view.RenderPhase.Append(RenderPhaseItem{
				Type:           &spriteRenderPhaseItem{},
				Draw:           drawSpriteBatch,
				SortValue:      sp.Transform.TranslateZ(),
				ExtractedIndex: uint32(idx),
			})
		}
	}
}

type metaSprites struct {
	Instances wgsl.InstanceWriter
	Buffer    *wgpu.Buffer
}

type spriteTextureBindGroupCache struct {
	BindGroups map[*Texture]*wgpu.BindGroup
}

func (c *spriteTextureBindGroupCache) Clear() {
	for _, bg := range c.BindGroups {
		bg.Release()
	}

	clear(c.BindGroups)
}

func (c *spriteTextureBindGroupCache) Add(texture *Texture, bindGroup *wgpu.BindGroup) {
	if c.BindGroups == nil {
		c.BindGroups = map[*Texture]*wgpu.BindGroup{}
	}

	c.BindGroups[texture] = bindGroup
}

func (c *spriteTextureBindGroupCache) Get(texture *Texture) (*wgpu.BindGroup, bool) {
	bindGroup, ok := c.BindGroups[texture]
	return bindGroup, ok
}

type spriteRenderPhaseItem struct{}

func prepareSpriteBindGroupsSystem(
	ctx *RenderContext,
	pipelineCache *PipelineCache,
	viewsQuery byke.Query[struct {
		_     byke.With[Camera]
		Phase RenderPhase
	}],
	sprites *ExtractedSprites,
	meta *metaSprites,
	bindGroups *spriteTextureBindGroupCache,
) {
	bindGroups.Clear()

	instances := &meta.Instances
	instances.Clear()

	for view := range viewsQuery.Items() {
		if view.Phase.IsEmpty() {
			continue
		}

		var current *RenderPhaseItem
		var currentSprite *ExtractedSprite

		for idx := range view.Phase.Len() {
			item := view.Phase.Get(idx)

			_, isSprite := item.Type.(*spriteRenderPhaseItem)

			if !isSprite {
				// not a sprite, end the current batch,
				current = nil
				currentSprite = nil
				continue
			}

			itemSprite := &sprites.Sprites[item.ExtractedIndex]

			//goland:noinspection GoMaybeNil
			if current == nil ||
				currentSprite.Texture != itemSprite.Texture ||
				currentSprite.CustomShader != itemSprite.CustomShader {

				// we begin a new sprite batch here
				current = item
				currentSprite = itemSprite

				// record begin of batch
				current.BatchBegin = uint32(instances.InstanceCount())
				current.BatchCount = 0

				// ensure bindgroup for image exists
				if _, ok := bindGroups.Get(itemSprite.Texture); !ok {
					bindGroups.Add(itemSprite.Texture,
						ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
							Label:  "Sprite Texture",
							Layout: pipelineCache.BindGroupLayout(layoutSpriteTextures),
							Entries: Sequential(
								BindingTextureView(itemSprite.Texture.TextureView),
								BindingSampler(itemSprite.Texture.Sampler),
							),
						}),
					)
				}
			}

			// write sprite vertex data
			writeSpriteInstanceValues(instances, itemSprite)
			current.BatchCount += 1
		}
	}

	// upload buffer to gpu
	instances.WriteTo(ctx, &meta.Buffer)
}

func writeSpriteInstanceValues(instances *wgsl.InstanceWriter, sp *ExtractedSprite) {
	// the size of one sprite instance in the wgpu instance buffer
	const instanceSize = 84

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
	transform := sp.Transform
	transform.ScaleAssign(sp.Size.Extend(1.0).XYZ())
	transform.TranslateAssign(-1*sp.Anchor.Vec2f[0]-0.5, sp.Anchor.Vec2f[1]-0.5, 0)

	var flags uint32
	if sp.Texture.Descriptor.Format == wgpu.TextureFormatR8Unorm {
		flags |= 1
	}

	instances.StartNew(instanceSize)

	// @location(0) i_affine_0: mat3<f32>,
	instances.AppendVec3f(transform.Column(0).Truncate())
	// @location(1) i_affine_1: mat3<f32>,
	instances.AppendVec3f(transform.Column(1).Truncate())
	// @location(2) i_affine_2: mat3<f32>,
	instances.AppendVec3f(transform.Column(2).Truncate())
	// @location(3) i_affine_3: mat3<f32>,
	instances.AppendVec3f(transform.Column(3).Truncate())
	// @location(4) i_uv_offset: vec2<f32>,
	instances.AppendVec2f(uvOffset)
	// @location(5) i_uv_scale: vec2<f32>,
	instances.AppendVec2f(uvScale)
	// @location(6) i_color: vec4<f32>,
	instances.AppendVec4f(sp.Color.ToVec())
	// @location(7) i_flags: u32,
	instances.AppendUint(flags)
}

type RenderTask struct {
	Pass *wgpu.RenderPassEncoder
	Item RenderPhaseItem
}

func drawSpriteBatch(world *byke.World, pass *wgpu.RenderPassEncoder, item RenderPhaseItem) (ok bool) {
	world.RunSystemWithInValue(drawSpriteBatchSystem, RenderTask{
		Pass: pass,
		Item: item,
	})

	return true
}

func drawSpriteBatchSystem(
	viewBindGroup ViewBindGroup,
	textureBindGroups spriteTextureBindGroupCache,
	pipelines Pipelines[renderSpritePipelineConfig],
	meta metaSprites,
	viewQuery ViewQuery[struct {
		ViewTarget         *ViewTarget
		ViewUniformsOffset DynamicOffset[ViewUniforms]
	}],
	extractedSprites ExtractedSprites,
	task byke.In[RenderTask],
) {
	view := viewQuery.Get()
	pass, item := task.Value.Pass, task.Value.Item

	sprite := &extractedSprites.Sprites[task.Value.Item.ExtractedIndex]

	pipeline := pipelines.Specialize(renderSpritePipelineConfig{
		Format:      view.ViewTarget.Format,
		SampleCount: view.ViewTarget.SampleCount,
		Shader:      sprite.CustomShader,
	})

	// get the bind group for the texture for this batch
	textureBindGroup, _ := textureBindGroups.Get(sprite.Texture)

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, viewBindGroup.BindGroup, []uint32{view.ViewUniformsOffset.Offset})
	pass.SetBindGroup(1, textureBindGroup, nil)
	pass.SetVertexBuffer(0, meta.Buffer, 0, wgpu.WholeSize)
	pass.Draw(6, item.BatchCount, 0, item.BatchBegin)
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

var layoutSpriteTextures = SequentialLayoutWithLabel("Spite Textures",
	BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
	BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
)
