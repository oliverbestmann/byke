package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func valueOr[T comparable](first, fallback T) T {
	var zero T
	if first != zero {
		return first
	}

	return fallback
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

type spriteRenderPhaseItem struct{}

type spriteTextureBindGroupCache struct {
	tickCache[*Texture, *wgpu.BindGroup]
}

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
	bindGroups.Tick()

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
	pipelines *PipelineCache,
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

var layoutSpriteTextures = SequentialLayoutWithLabel("Spite Textures",
	BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false),
	BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering),
)
