package byke2d

import (
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"slices"
	"unicode"

	"github.com/go-text/render"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/segmenter"
	"github.com/go-text/typesetting/shaping"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/webgpu/wgpu"
	"golang.org/x/image/math/fixed"
)

var _ = byke.ValidateComponent[Text]()
var _ = byke.ValidateComponent[Font]()
var _ = byke.ValidateComponent[textCache]()

var sharedShaper shaping.HarfbuzzShaper
var sharedSegmenter shaping.Segmenter

type Text struct {
	byke.ComparableComponent[Text]
	Text  string
	Size  float32
	Color Color
}

func (Text) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
		DefaultFont(),
		InheritVisibility,
		AnchorCenter,
		renderLayerZero,
		textCache{},
	}
}

type Font struct {
	byke.ComparableComponent[Font]
	Faces Faces
}

type textCache struct {
	byke.Component[textCache]
	Text   []rune
	Layout textLayout
	Size   float32
}

func cacheTextSystem(
	ctx *RenderContext,
	textQuery byke.Query[struct {
		_     byke.Changed[Text]
		Text  Text
		Font  Font
		Cache *textCache
	}],
) {
	for item := range textQuery.Items() {
		text := []rune(item.Text.Text)
		layout := layoutText(text, item.Font.Faces, item.Text.Size)

		// cache values
		item.Cache.Layout = layout
		item.Cache.Text = text
		item.Cache.Size = item.Text.Size

		for _, line := range layout.Lines {
			for idx, output := range line.Outputs {
				cacheGlyphs(ctx, item.Text.Size, line.Inputs[idx], output)
			}
		}
	}
}

type renderTextValue struct {
	Entity          byke.EntityId
	Text            Text
	Anchor          Anchor
	Font            Font
	RenderLayers    RenderLayers
	GlobalTransform GlobalTransform
	Cache           textCache
}

func renderTextSystem(
	textQuery byke.Query[renderTextValue],
	sprites *ExtractedSprites,
) {
	for item := range textQuery.Items() {
		text := item.Cache.Text
		layout := item.Cache.Layout
		textSize := item.Cache.Size

		// calculate origin for the text
		offset := layout.Size.Mul(item.Anchor.Mul(glm.Vec2f{-1, 1}).Add(glm.Vec2f{-0.5, -0.5}))

		var posX float32
		var posY float32

		for _, line := range slices.Backward(layout.Lines) {

			for _, output := range line.Outputs {
				for _, glyph := range output.Glyphs {
					x := posX + floatOf(glyph.XOffset) + offset[0]
					y := posY + floatOf(glyph.YOffset) + offset[1]

					// advance to the next glyph
					posX += floatOf(glyph.Advance)

					if unicode.IsSpace(text[glyph.TextIndex()]) {
						// no need to spawn sprites for whitespace
						continue
					}

					glyphTexture, ok := glyphCache.Lookup(output.Face, textSize, glyph.GlyphID)
					if !ok {
						continue
					}

					// offset position by the texture offset
					x += glyphTexture.Offset[0]
					y -= glyphTexture.Offset[1]

					// rectangle within the texture
					rect := glm.Rectf{
						Min: glyphTexture.Rectangle.Min.ToVec2f(),
						Max: glyphTexture.Rectangle.Max.ToVec2f(),
					}

					transform := item.GlobalTransform.Mul(TransformFromXY(x, y))

					sprites.Sprites = append(sprites.Sprites, ExtractedSprite{
						Texture:      glyphTexture.Texture,
						Color:        item.Text.Color,
						RenderLayers: item.RenderLayers,
						Transform:    transform.Affine,
						Rect:         rect,
						Size:         rect.Size(),
						Anchor:       *AnchorBottomLeft,
					})
				}
			}

			posX = 0
			posY += line.Size[1]
		}
	}
}

func floatOf(value fixed.Int26_6) float32 {
	return float32(value) / 64.0
}

type textLayout struct {
	Lines []lineLayout
	Size  glm.Vec2f
}

type lineLayout struct {
	Inputs  []shaping.Input
	Outputs []shaping.Output
	Size    glm.Vec2f
}

func layoutText(text []rune, faces Faces, fontSize float32) textLayout {
	defer puffin.NewScope("text.Layout").End()

	lines := splitTextToLines(text)

	var size glm.Vec2f
	var layouts []lineLayout

	for _, line := range lines {
		layout := layoutLine(text, line, faces, fontSize)
		layouts = append(layouts, layout)

		size[0] = max(size[0], layout.Size[0])
		size[1] += layout.Size[1]
	}

	return textLayout{Lines: layouts, Size: size}
}

func splitTextToLines(text []rune) []lineIndices {
	var lines []lineIndices

	var seg segmenter.Segmenter
	seg.Init(text)

	lineIter := seg.LineIterator()

	var lineStart = -1

	for lineIter.Next() {
		line := lineIter.Line()

		if lineStart < 0 {
			lineStart = line.Offset
		}

		if line.IsMandatoryBreak {
			end := line.Offset + len(line.Text)
			lines = append(lines, lineIndices{Offset: lineStart, End: end})
			lineStart = -1
		}
	}

	if lineStart >= 0 {
		lines = append(lines, lineIndices{Offset: lineStart, End: len(text)})
	}
	return lines
}

type lineIndices struct {
	Offset int
	End    int
}

func layoutLine(text []rune, line lineIndices, faces Faces, fontSize float32) lineLayout {
	input := shaping.Input{
		Text:      text,
		RunStart:  line.Offset,
		RunEnd:    line.End,
		Direction: di.DirectionLTR,
		Size:      fixed.Int26_6(fontSize * float32(1<<6)),
		FontFeatures: []shaping.FontFeature{
			{
				Tag:   opentype.MustNewTag("liga"),
				Value: 1,
			},
		},
	}

	var size glm.Vec2f

	var inputs []shaping.Input
	var outputs []shaping.Output

	for _, input := range sharedSegmenter.Split(input, faces) {
		output := sharedShaper.Shape(input)

		for idx := range output.Glyphs {
			glyph := &output.Glyphs[idx]
			size[0] += floatOf(glyph.Advance)
		}

		size[1] = max(size[1], floatOf(output.LineBounds.LineThickness()))

		inputs = append(inputs, input)
		outputs = append(outputs, output)
	}

	return lineLayout{
		Inputs:  inputs,
		Outputs: outputs,
		Size:    size,
	}
}

func cacheGlyphs(ctx *RenderContext, fontSize float32, input shaping.Input, output shaping.Output) {
	defer puffin.NewScope("text.CacheGlyphs").End()

	renderer := render.Renderer{
		Color:    color.White,
		FontSize: fontSize,
		PixScale: 1.0,
	}

	height := (output.GlyphBounds.Ascent - output.GlyphBounds.Descent).Ceil()

	for _, glyph := range output.Glyphs {
		_, ok := glyphCache.Lookup(output.Face, fontSize, glyph.GlyphID)
		if ok {
			// already cached
			continue
		}

		width := glyph.Width.Ceil() + glyph.XBearing.Ceil()

		if width <= 0 || height <= 0 {
			continue
		}

		// shallow copy of output, we want to modify the slice of glyphs
		output := output
		output.Glyphs = []shaping.Glyph{glyph}

		img := image.NewNRGBA(image.Rectangle{Max: image.Point{X: width + 8, Y: height * 2}})

		startY := output.LineBounds.Ascent.Ceil()
		startX := 0

		renderer.DrawShapedRunAt(output, img, startX, startY)

		entry := glyphCache.Store(ctx, output.Face, fontSize, glyph.GlyphID, img)

		slog.Info("Cache glyph",
			slog.Int("glyph", int(input.Text[glyph.TextIndex()])),
			slog.Any("size", entry.Rectangle.Size()),
			slog.Any("offset", entry.Offset),
			slog.Int("glyphCount", glyphCache.GlyphCount()),
		)
	}
}

type GlyphTexture struct {
	Texture   *Texture
	Rectangle glm.Rectu
	Offset    glm.Vec2f
}

func (g *GlyphTexture) IsValid() bool {
	return g.Texture != nil
}

type GlyphCache struct {
	Allocator TextureAtlasAllocator
	cache     map[glyphCacheKey]GlyphTexture
}

func NewFontCache() *GlyphCache {
	return &GlyphCache{
		Allocator: TextureAtlasAllocator{
			TextureFormat: wgpu.TextureFormatR8Unorm,
		},

		cache: map[glyphCacheKey]GlyphTexture{},
	}
}

func (c *GlyphCache) Lookup(face *font.Face, fontSize float32, glyphId font.GID) (GlyphTexture, bool) {
	key := glyphCacheKey{
		FontFace: face,
		Size:     fontSize,
		GlyphId:  glyphId,
	}

	if cached, ok := c.cache[key]; ok {
		return cached, true
	}

	return GlyphTexture{}, false
}

func (c *GlyphCache) Store(ctx *RenderContext, face *font.Face, fontSize float32, glyphId font.GID, src *image.NRGBA) GlyphTexture {
	rectGlyph := regionOfInterest(src)

	var padding = 1

	// allocate a buffer to hold the non-transparent region
	// add padding to the glyphs size
	rgba := image.NewAlpha(image.Rect(0, 0, rectGlyph.Dx()+2*padding, rectGlyph.Dy()+2*padding))

	// pick the sub image that the glyph will be placed at
	rgbaGlyph := rgba.Bounds().Inset(padding)

	// copy the glyph over
	draw.Draw(rgba, rgbaGlyph.Bounds(), src, rectGlyph.Min, draw.Src)

	// DEBUG: fill texture with color for debugging the region
	// debug := color.RGBA{R: 255, G: 255, B: 255, A: 64}
	// draw.Draw(rgba, rgbaGlyph.Bounds(), image.NewUniform(debug), image.Point{}, draw.Over)

	// allocate a texture of the required size
	texWidth, texHeight := uint32(rgba.Bounds().Dx()), uint32(rgba.Bounds().Dy())
	tex, region := c.Allocator.Allocate(ctx, texWidth, texHeight)

	// upload the glyph data to the sub rectangle
	tex.WritePixelsToRect(ctx, WritePixelsOptions{
		Stride: uint32(rgba.Stride),
		Pixels: rgba.Pix,
		Region: region,
	})

	key := glyphCacheKey{
		FontFace: face,
		Size:     fontSize,
		GlyphId:  glyphId,
	}

	result := GlyphTexture{
		Texture:   tex,
		Rectangle: region,
		Offset: glm.Vec2f{
			// move the glyph origin back to the
			// origin of the provided src image
			float32(rectGlyph.Min.X) - float32(padding),
			float32(rectGlyph.Max.Y) - float32(padding),
		},
	}

	c.cache[key] = result

	return result
}

func (c *GlyphCache) GlyphCount() int {
	return len(c.cache)
}

func regionOfInterest(src *image.NRGBA) image.Rectangle {
	rect := src.Bounds()

	// walk through each row of the image and skip empty rows
	for ; rect.Min.Y < rect.Max.Y; rect.Min.Y++ {
		if !rowIsTransparent(src, rect.Min.Y) {
			break
		}
	}

	for ; rect.Max.Y > rect.Min.Y; rect.Max.Y-- {
		if !rowIsTransparent(src, rect.Max.Y-1) {
			break
		}
	}

	// walk through columns
	for ; rect.Min.X < rect.Max.X; rect.Min.X++ {
		if !columnIsTransparent(src, rect.Min.X) {
			break
		}
	}

	for ; rect.Max.X > rect.Min.X; rect.Max.X-- {
		if !columnIsTransparent(src, rect.Max.X-1) {
			break
		}
	}

	if rect.Empty() {
		// return a non empty rect^
		return image.Rect(0, 0, 1, 1)
	}

	return rect
}

func rowIsTransparent(src *image.NRGBA, y int) bool {
	width := src.Bounds().Dx()

	for x := range width {
		if src.NRGBAAt(x, y).A > 0 {
			return false
		}
	}

	return true
}

func columnIsTransparent(src *image.NRGBA, x int) bool {
	height := src.Bounds().Dy()

	for y := range height {
		if src.NRGBAAt(x, y).A > 0 {
			return false
		}
	}

	return true
}

type glyphCacheKey struct {
	FontFace *font.Face
	Size     float32
	GlyphId  font.GID
}

var glyphCache = NewFontCache()
