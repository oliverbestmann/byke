package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

var _ = byke.ValidateComponent[Sprite]()
var _ = byke.ValidateComponent[Anchor]()

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
