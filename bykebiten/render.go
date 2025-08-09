package bykebiten

import (
	"image"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
)

var whiteImage = sync.OnceValue(func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
})

var _ = byke.ValidateComponent[Layer]()
var _ = byke.ValidateComponent[ColorTint]()
var _ = byke.ValidateComponent[Anchor]()
var _ = byke.ValidateComponent[BBox]()

type Layer struct {
	byke.ComparableComponent[Layer]
	Z float64
}

type Filter struct {
	byke.Component[Filter]
	ebiten.Filter
}

type Blend struct {
	byke.Component[Blend]
	ebiten.Blend
}

type Anchor struct {
	byke.ComparableComponent[Anchor]
	gm.Vec
}

var (
	AnchorTopLeft      = &Anchor{Vec: gm.Vec{}}
	AnchorTopCenter    = &Anchor{Vec: gm.Vec{X: 0.5}}
	AnchorTopRight     = &Anchor{Vec: gm.Vec{X: 1.0}}
	AnchorCenterLeft   = &Anchor{Vec: gm.Vec{X: 0, Y: 0.5}}
	AnchorCenter       = &Anchor{Vec: gm.Vec{X: 0.5, Y: 0.5}}
	AnchorCenterRight  = &Anchor{Vec: gm.Vec{X: 1.0, Y: 0.5}}
	AnchorBottomLeft   = &Anchor{Vec: gm.Vec{Y: 1.0}}
	AnchorBottomCenter = &Anchor{Vec: gm.Vec{X: 0.5, Y: 1.0}}
	AnchorBottomRight  = &Anchor{Vec: gm.Vec{X: 1.0, Y: 1.0}}
)

type ColorTint struct {
	byke.ComparableComponent[ColorTint]
	color.Color
}

type screenRenderTarget struct {
	Image *ebiten.Image
}

type BBox struct {
	byke.ComparableComponent[BBox]
	gm.Rect

	// the bounding box might not reflect the actual source, e.g. for sprites
	// that have a custom size set. ToSourceScale describes the factor that
	// the Rect would need to be multiplied with to get the sources size.
	ToSourceScale gm.Vec
}

var commonRenderComponents = []byke.ErasedComponent{
	Transform{Scale: gm.VecOne},
	Layer{},
	ColorTint{Color: color.White},
	BBox{},
	InheritVisibility,
	ComputedVisibility{},
}

type renderCommonValues struct {
	BBox               BBox
	ColorTint          ColorTint
	Layer              Layer
	Transform          GlobalTransform
	RenderLayers       byke.Option[RenderLayers]
	ComputedVisibility ComputedVisibility
}

type hasCommonValues interface {
	commonValues() *renderCommonValues
	needsTranslate() bool
}

type renderSpriteValue struct {
	Common       renderCommonValues
	Sprite       Sprite
	Filter       Filter
	Blend        Blend
	TileIndex    byke.Option[TileIndex]
	TileCache    byke.Option[tileCache]
	Shader       byke.Option[Shader]
	ShaderInputs byke.Option[ShaderInput]
}

func (r *renderSpriteValue) commonValues() *renderCommonValues {
	return &r.Common
}

func (*renderSpriteValue) needsTranslate() bool {
	return true
}

type renderTextValue struct {
	Common renderCommonValues
	Text   Text
	Face   TextFace
	Filter Filter
	Blend  Blend
}

func (r *renderTextValue) commonValues() *renderCommonValues {
	return &r.Common
}

func (*renderTextValue) needsTranslate() bool {
	return true
}

type renderVectorValue struct {
	Common renderCommonValues
	Path   Path
	Blend  Blend
	Fill   byke.Option[Fill]
	Stroke byke.Option[Stroke]
}

func (r *renderVectorValue) commonValues() *renderCommonValues {
	return &r.Common
}

func (*renderVectorValue) needsTranslate() bool {
	return false
}

type renderMeshValue struct {
	Common       renderCommonValues
	Mesh         Mesh
	Blend        Blend
	Filter       Filter
	Shader       byke.Option[Shader]
	ShaderInputs byke.Option[ShaderInput]
}

func (r *renderMeshValue) commonValues() *renderCommonValues {
	return &r.Common
}

func (*renderMeshValue) needsTranslate() bool {
	return false
}

type renderCache struct {
	Cameras []cameraValue

	Sprites []renderSpriteValue
	Texts   []renderTextValue
	Vectors []renderVectorValue
	Meshes  []renderMeshValue

	TempPath vector.Path

	// scratch space for vertex transformations
	vertexCache []ebiten.Vertex

	all []hasCommonValues
}

type cameraValue struct {
	Camera       Camera
	Transform    GlobalTransform
	Projection   OrthographicProjection
	RenderLayers byke.Option[RenderLayers]
}

func renderSystem(
	screen screenRenderTarget,
	camerasQuery byke.Query[cameraValue],
	spritesQuery byke.Query[renderSpriteValue],
	textsQuery byke.Query[renderTextValue],
	vectorsQuery byke.Query[renderVectorValue],
	meshesQuery byke.Query[renderMeshValue],
	cache *byke.Local[renderCache],
) {
	c := &cache.Value

	defer func() {
		clear(c.Sprites)
		clear(c.Texts)
		clear(c.Vectors)
		clear(c.all)
	}()

	// re-use the slices and collect all values
	c.Cameras = camerasQuery.AppendTo(c.Cameras[:0])
	c.Sprites = spritesQuery.AppendTo(c.Sprites[:0])
	c.Texts = textsQuery.AppendTo(c.Texts[:0])
	c.Vectors = vectorsQuery.AppendTo(c.Vectors[:0])
	c.Meshes = meshesQuery.AppendTo(c.Meshes[:0])

	items := c.Items()

	// sort sprites by layer
	slices.SortFunc(items, func(a, b hasCommonValues) int {
		return compareZ(a.commonValues(), b.commonValues())
	})

	// sort cameras
	slices.SortFunc(c.Cameras, func(a, b cameraValue) int {
		return a.Camera.Order - b.Camera.Order
	})

	for _, camera := range c.Cameras {
		// get the target to render to
		renderTarget := camera.Camera.Target.Image
		if renderTarget == nil {
			renderTarget = screen.Image
		}

		// render all items that are part of this camera
		renderItems(c, renderTarget, camera, items)
	}
}

func renderItems(c *renderCache, screen *ebiten.Image, camera cameraValue, items []hasCommonValues) {
	var crl RenderLayers = camera.RenderLayers.Or(renderLayerZero)

	screenSize := imageSizeOf(screen)

	// get the sub viewport of the image if needed
	if sub := camera.Camera.SubCameraView; sub != nil {
		rect := gm.Rect{
			Min: sub.Min.MulEach(screenSize),
			Max: sub.Max.MulEach(screenSize),
		}

		screen = screen.SubImage(rect.ToImageRectangle()).(*ebiten.Image)
		screenSize = imageSizeOf(screen)
	}

	// if we have a clear color, clear the image
	if cc := camera.Camera.ClearColor; cc != nil {
		screen.Fill(cc)
	}

	cameraWorldToScreen := CalculateWorldToScreenTransform(camera.Projection, camera.Transform, screenSize)

	var toScreen ebiten.GeoM
	toScreen.SetElement(0, 0, cameraWorldToScreen.Matrix.XAxis.X)
	toScreen.SetElement(0, 1, cameraWorldToScreen.Matrix.XAxis.Y)
	toScreen.SetElement(0, 2, cameraWorldToScreen.Translation.X)
	toScreen.SetElement(1, 0, cameraWorldToScreen.Matrix.YAxis.X)
	toScreen.SetElement(1, 1, cameraWorldToScreen.Matrix.YAxis.Y)
	toScreen.SetElement(1, 2, cameraWorldToScreen.Translation.Y)

	for _, item := range items {
		common := item.commonValues()

		if !crl.Intersects(common.RenderLayers.Or(renderLayerZero)) {
			continue
		}

		if !item.commonValues().ComputedVisibility.Visible {
			continue
		}

		var g ebiten.GeoM

		// get transformation
		tr := common.Transform

		// custom scale, e.g. derived from sprites CustomSize property
		toSourceScale := common.BBox.ToSourceScale
		if toSourceScale != gm.VecOne {
			g.Scale(1/toSourceScale.X, 1/toSourceScale.Y)
		}

		if item.needsTranslate() {
			// transform by offset. Need to multiply by the sign of source scale as that might
			// flip the direction the origin translation need to be applied
			origin := common.BBox.Min
			g.Translate(origin.X*signOf(toSourceScale.X), origin.Y*signOf(toSourceScale.Y))
		}

		if tr.Scale != gm.VecOne {
			// apply custom size based on transform
			g.Scale(tr.Scale.X, tr.Scale.Y)
		}

		if tr.Rotation != 0 {
			// apply rotation
			g.Rotate(float64(tr.Rotation))
		}

		// move to target position
		g.Translate(tr.Translation.X, tr.Translation.Y)

		g.Concat(toScreen)

		// apply color
		var colorScale ebiten.ColorScale
		colorScale.Scale(common.ColorTint.PremultipliedValues())

		switch item := item.(type) {
		case *renderSpriteValue:
			image := item.Sprite.Image
			if tileCache_, ok := item.TileCache.Get(); ok {
				var tileCache tileCache = tileCache_

				idx := item.TileIndex.OrZero().Index
				image = tileCache.Tiles[idx%len(tileCache.Tiles)]
			}

			if shader_, ok := item.Shader.Get(); ok {
				var shader *ebiten.Shader = shader_.Shader
				var inputs ShaderInput = item.ShaderInputs.OrZero()

				imageSize := intImageSizeOf(image)

				screen.DrawRectShader(imageSize.X, imageSize.Y, shader, &ebiten.DrawRectShaderOptions{
					GeoM:       g,
					ColorScale: colorScale,
					Blend:      item.Blend.Blend,
					Uniforms:   inputs.Uniforms,
					Images: [4]*ebiten.Image{
						image,
						inputs.Images[1],
						inputs.Images[2],
						inputs.Images[3],
					},
				})

			} else {
				var op ebiten.DrawImageOptions
				op.GeoM = g
				op.ColorScale = colorScale
				op.Blend = item.Blend.Blend
				op.Filter = item.Filter.Filter
				screen.DrawImage(image, &op)
			}

		case *renderTextValue:
			var op text.DrawOptions
			op.GeoM = g
			op.ColorScale = colorScale
			op.Blend = item.Blend.Blend
			op.Filter = item.Filter.Filter
			op.LineSpacing = item.Face.Metrics().VLineGap
			text.Draw(screen, item.Text.Text, item.Face, &op)

		case *renderVectorValue:
			path := &c.TempPath
			path.Reset()
			path.AddPath(item.Path.inner(), &vector.AddPathOptions{GeoM: g})

			if fill_, ok := item.Fill.Get(); ok {
				var fill Fill = fill_

				dop := &vector.DrawPathOptions{}
				dop.AntiAlias = fill.Antialias
				dop.Blend = item.Blend.Blend
				dop.ColorScale.Scale(fill.Color.PremultipliedValues())
				vector.FillPath(screen, path, &vector.FillOptions{FillRule: fill.Rule}, dop)
			}

			if stroke_, ok := item.Stroke.Get(); ok {
				var stroke Stroke = stroke_

				// estimate stroke
				origin := gm.VecOf(g.Apply(0, 0))
				strokeWidth := gm.VecOf(g.Apply(1, 0)).DistanceTo(origin) * stroke.Width

				sop := &vector.StrokeOptions{
					Width:      float32(strokeWidth),
					LineCap:    stroke.LineCap,
					LineJoin:   stroke.LineJoin,
					MiterLimit: float32(stroke.MiterLimit),
				}

				dop := &vector.DrawPathOptions{}
				dop.AntiAlias = stroke.Antialias
				dop.Blend = item.Blend.Blend
				dop.ColorScale.Scale(stroke.Color.PremultipliedValues())
				vector.StrokePath(screen, path, sop, dop)
			}

		case *renderMeshValue:
			colorTint := item.Common.ColorTint.Color
			vertices := transformVertices(item.Mesh.Vertices, g, colorTint)

			if shader_, ok := item.Shader.Get(); ok {
				var shader *ebiten.Shader = shader_.Shader
				var inputs ShaderInput = item.ShaderInputs.OrZero()

				screen.DrawTrianglesShader32(vertices, item.Mesh.Indices, shader, &ebiten.DrawTrianglesShaderOptions{
					Blend:    item.Blend.Blend,
					Uniforms: inputs.Uniforms,
					Images:   inputs.Images,
				})

			} else {
				screen.DrawTriangles32(vertices, item.Mesh.Indices, whiteImage(), &ebiten.DrawTrianglesOptions{
					Blend:  item.Blend.Blend,
					Filter: item.Filter.Filter,
				})
			}
		}
	}
}

func CalculateWorldToScreenTransform(projection OrthographicProjection, cameraTransform GlobalTransform, screenSize gm.Vec) gm.Affine {
	// calculate the cameras viewport size in world units
	viewportSizeInWorld := projection.ScalingMode.
		ViewportSize(screenSize.X, screenSize.Y).
		Mul(projection.Scale).
		MulEach(cameraTransform.Scale)

	toScreen := gm.IdentityAffine()

	// and the offset from the center of the viewport in world units
	viewportOffsetInWorld := projection.ViewportOrigin.MulEach(viewportSizeInWorld)

	scaleWorldToScreen := screenSize.DivEach(viewportSizeInWorld)

	// scale the viewport
	toScreen = toScreen.Scale(scaleWorldToScreen)

	// move the viewport
	toScreen = toScreen.Translate(viewportOffsetInWorld)

	// now rotate everything around that point
	toScreen = toScreen.Rotate(-cameraTransform.Rotation)

	// move the camera to the target position in world space
	toScreen = toScreen.Translate(cameraTransform.Translation.Mul(-1))
	return toScreen
}

func (c *renderCache) Items() []hasCommonValues {
	c.all = c.all[:0]

	for idx := range c.Sprites {
		c.all = append(c.all, &c.Sprites[idx])
	}

	for idx := range c.Texts {
		c.all = append(c.all, &c.Texts[idx])
	}

	for idx := range c.Vectors {
		c.all = append(c.all, &c.Vectors[idx])
	}

	for idx := range c.Meshes {
		c.all = append(c.all, &c.Meshes[idx])
	}

	return c.all
}

func compareZ(a, b *renderCommonValues) int {
	switch {
	case a.Layer.Z < b.Layer.Z:
		return -1
	case a.Layer.Z > b.Layer.Z:
		return 1
	default:
		return 0
	}
}

func signOf(x float64) float64 {
	if x < 0 {
		return -1
	} else {
		return 1
	}
}

var scratchVertices []ebiten.Vertex

func transformVertices(vertices []Vertex, g ebiten.GeoM, tint color.Color) []ebiten.Vertex {
	scratchVertices = append(scratchVertices[:0], vertices...)

	for idx := range scratchVertices {
		x, y := g.Apply(float64(scratchVertices[idx].DstX), float64(scratchVertices[idx].DstY))
		scratchVertices[idx].DstX = float32(x)
		scratchVertices[idx].DstY = float32(y)

	}

	if !tint.IsIdentity() {
		r, g, b, a := tint.PremultipliedValues()

		for idx := range scratchVertices {
			scratchVertices[idx].ColorR *= r
			scratchVertices[idx].ColorG *= g
			scratchVertices[idx].ColorB *= b
			scratchVertices[idx].ColorA *= a
		}
	}

	return scratchVertices
}
