package byke2d

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log/slog"

	"github.com/go-text/render"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

var _ = byke.ValidateComponent[Text]()

var defaultFontFace *font.Face

var fontShaper shaping.HarfbuzzShaper

func init() {
	f, _ := font.ParseTTF(bytes.NewReader(goregular.TTF))
	defaultFontFace = f
}

type Text struct {
	byke.ComparableComponent[Text]
	Text  string
	Size  float32
	Color wx.Color
}

func (Text) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
		InheritVisibility,
		AnchorBottomLeft,
	}
}

type Glyph struct {
	byke.ImmutableComponent[Glyph]
	glyph     shaping.Glyph
	charValue rune
}

type renderTextValue struct {
	_ byke.Changed[Text]

	Entity byke.EntityId
	Text   Text
	Anchor Anchor

	Children byke.Option[byke.Children]
}

func renderTextSystem(ctx *RenderContext,
	commands *byke.Commands,
	textQuery byke.Query[renderTextValue],
	glyphSpriteQuery byke.Query[struct {
		_ byke.With[byke.ChildOf]
		_ byke.With[Glyph]
		_ byke.With[Sprite]
	}],
) {
	for item := range textQuery.Items() {
		if children, ok := item.Children.Get(); ok {
			for _, child := range children.Children() {
				_, ok := glyphSpriteQuery.Get(child)
				if ok {
					commands.Entity(child).Despawn()
				}
			}
		}

		text := []rune(item.Text.Text)
		output := prepareTextGlyphs(ctx, text, item.Text.Size)

		var posX float32
		var posY float32

		offset := output.Size.Mul(item.Anchor.Vec2f)

		for _, glyph := range output.Glyphs {
			x := posX + floatOf(glyph.XOffset) - offset[0]
			// y is still a little off
			y := posY + offset[1] + floatOf(glyph.YOffset) + floatOf(output.LineBounds.Ascent) - floatOf(output.LineBounds.Descent)

			posX += floatOf(glyph.Advance)

			texture, region, ok := glyphCache.Lookup(output.Face, item.Text.Size, glyph.GlyphID)
			if !ok {
				continue
			}

			customSize := glm.Vec2f{
				floatOf(glyph.Width + glyph.XBearing),
				floatOf(output.LineBounds.Ascent - output.LineBounds.Descent),
			}

			sprite := Sprite{
				Texture:    texture,
				Color:      item.Text.Color,
				CustomSize: Some(customSize),
			}

			commands.Spawn(
				byke.ChildOf{Parent: item.Entity},
				TransformFromXY(x, y),
				Glyph{glyph: glyph, charValue: text[glyph.TextIndex()]},
				TextureAtlas{Layout: TextureAtlasLayoutFromRect(region)},
				AnchorBottomLeft,
				sprite,
			)

		}
	}
}

func floatOf(value fixed.Int26_6) float32 {
	return float32(value) / 64.0
}

type layoutOutput struct {
	shaping.Output
	Size glm.Vec2f
}

func prepareTextGlyphs(ctx *RenderContext, text []rune, fontSize float32) layoutOutput {
	// Input configuration
	input := shaping.Input{
		Text:      text,
		RunStart:  0,
		RunEnd:    len(text),
		Direction: di.DirectionLTR,
		Face:      defaultFontFace,
		Size:      fixed.Int26_6(fontSize * float32(1<<6)),
		Script:    language.Latin,
		Language:  language.DefaultLanguage(),
	}

	output := fontShaper.Shape(input)

	renderer := render.Renderer{
		Color:    color.White,
		FontSize: fontSize,
		PixScale: 1.0,
	}

	height := (output.GlyphBounds.Ascent - output.GlyphBounds.Descent).Ceil()

	for _, glyph := range output.Glyphs {
		_, _, ok := glyphCache.Lookup(output.Face, fontSize, glyph.GlyphID)
		if ok {
			// already cached
			continue
		}

		width := glyph.Width.Ceil() + glyph.XBearing.Ceil()

		if width <= 0 || height <= 0 {
			continue
		}

		slog.Info("Cache glyph",
			slog.Int("glyph", int(input.Text[glyph.TextIndex()])),
			slog.Int("width", width),
			slog.Int("height", height),
		)

		output := output
		output.Glyphs = []shaping.Glyph{glyph}

		img := image.NewNRGBA(image.Rectangle{Max: image.Point{X: width, Y: height}})

		startY := output.LineBounds.Ascent.Ceil()
		startX := 0
		renderer.DrawShapedRunAt(output, img, startX, startY)

		glyphCache.Store(ctx, output.Face, fontSize, glyph.GlyphID, img)
	}

	size := glm.Vec2f{0, float32(height)}

	for idx := range output.Glyphs {
		glyph := &output.Glyphs[idx]

		size[0] += floatOf(glyph.Advance)
	}

	return layoutOutput{Output: output, Size: size}
}

type TextureAtlasAllocator struct {
	SamplerConfig SamplerConfig
	textures      []cacheTexture
}

func (t *TextureAtlasAllocator) Allocate(ctx *RenderContext, width, height uint32) (*Texture, wx.Rectangle2u) {
	tex, slice := t.findSlice(ctx, height, width)

	// extract the target region
	region := wx.RectangleFromXYWH(slice.NextX, slice.Y, width, height)

	// consume space in this new slice
	slice.NextX += width
	slice.AvailableWidth -= width

	return tex, region
}

func (t *TextureAtlasAllocator) findSlice(ctx *RenderContext, height, width uint32) (*Texture, *cacheTextureSlice) {
	// find a matching slice that still has space
	for _, tex := range t.textures {
		for idx := range tex.Slices {
			slice := &tex.Slices[idx]
			if slice.Height == height && slice.AvailableWidth >= width {
				return tex.Texture, slice
			}
		}
	}

	// find the first texture that still has room
	for idx := range t.textures {
		tex := &t.textures[idx]

		if tex.Available >= height {
			// start a new slice
			slice := cacheTextureSlice{
				Y:              tex.NextY,
				AvailableWidth: tex.Texture.Width(),
				Height:         height,
			}

			tex.Slices = append(tex.Slices, slice)

			// remove space width
			tex.Available -= height
			tex.NextY += height

			// return reference to the new slice
			refSlice := &tex.Slices[len(tex.Slices)-1]
			return tex.Texture, refSlice
		}
	}

	// allocate a new texture and try again
	texture := NewTexture(ctx, NewTextureOptions{
		SamplerConfig: t.SamplerConfig,
		Label:         "TextureAtlas",
		Format:        wgpu.TextureFormatBGRA8UnormSrgb,
		Width:         2048,
		Height:        2048,
	})

	t.textures = append(t.textures, cacheTexture{
		Texture:   texture,
		Available: texture.Width(),
	})

	if tex, slice := t.findSlice(ctx, height, width); tex != nil {
		return tex, slice
	}

	// still no space?
	panic(fmt.Errorf("failed to allocate slice for height %d, width %d", height, width))
}

type FontCache struct {
	Allocator TextureAtlasAllocator
	cache     map[glyphCacheKey]cachedSubTexture
}

func NewFontCache() *FontCache {
	return &FontCache{
		cache: map[glyphCacheKey]cachedSubTexture{},
	}
}

func (c *FontCache) Lookup(face *font.Face, fontSize float32, glyphId font.GID) (*Texture, wx.Rectangle2u, bool) {
	key := glyphCacheKey{
		FontFace: face,
		Size:     uint32(fontSize + 0.5),
		GlyphId:  glyphId,
	}

	if cached, ok := c.cache[key]; ok {
		return cached.Texture, cached.Region, true
	}

	return nil, wx.Rectangle2u{}, false
}

func (c *FontCache) Store(ctx *RenderContext, face *font.Face, fontSize float32, glyphId font.GID, src image.Image) (*Texture, wx.Rectangle2u) {
	width := src.Bounds().Dx()
	height := src.Bounds().Dy()

	var padding uint32 = 1

	tex, region := c.Allocator.Allocate(ctx, uint32(width)+2*padding, uint32(height)+2*padding)

	rgba := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), src, image.Point{}, draw.Src)

	regionInner := wx.RectangleFromPoints(
		region.Min.Add(glm.Vec2u{padding, padding}),
		region.Max.Sub(glm.Vec2u{padding, padding}),
	)

	tex.WritePixelsToRect(ctx, WritePixelsOptions{
		Pixels: rgba.Pix,
		Region: regionInner,
	})

	key := glyphCacheKey{
		FontFace: face,
		Size:     uint32(fontSize + 0.5),
		GlyphId:  glyphId,
	}

	c.cache[key] = cachedSubTexture{
		Texture: tex,
		Region:  regionInner,
	}

	return tex, regionInner
}

type cacheTexture struct {
	Texture   *Texture
	Slices    []cacheTextureSlice
	NextY     uint32
	Available uint32
}

type cacheTextureSlice struct {
	Y, Height             uint32
	NextX, AvailableWidth uint32
}

type cachedSubTexture struct {
	Texture *Texture
	Region  wx.Rectangle2u
}

type glyphCacheKey struct {
	FontFace *font.Face
	Size     uint32
	GlyphId  font.GID
}

var glyphCache = NewFontCache()
