package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/internal/query"
)

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
			RenderLayers: item.RenderLayers.Or(renderLayerZero),
			Transform:    item.Transform.Value.Affine,
			Anchor:       item.Anchor,
			Rect:         rect,
		})
	}
}
