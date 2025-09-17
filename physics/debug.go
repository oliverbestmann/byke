package physics

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	b2 "github.com/oliverbestmann/box2d-go"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
)

func debugSystem(
	world b2World,
	renderTarget bykebiten.DefaultRenderTarget,
	screenSize bykebiten.ScreenSize,
	cameraQuery byke.Query[struct {
		Projection bykebiten.OrthographicProjection
		Transform  bykebiten.GlobalTransform
	}],
) {
	screen := renderTarget.Image

	item, _ := cameraQuery.Single()
	toScreen := bykebiten.CalculateWorldToScreenTransform(item.Projection, item.Transform, screenSize.Vec)

	var draw b2.DebugDraw

	draw.DrawJoints = true
	draw.DrawShapes = true
	draw.DrawJointExtras = false
	draw.DrawBounds = true
	draw.DrawMass = false
	draw.DrawBodyNames = false
	draw.DrawGraphColors = false
	draw.DrawContacts = false
	draw.DrawContactNormals = true
	draw.DrawContactImpulses = true
	draw.DrawContactFeatures = false
	draw.DrawFrictionImpulses = false
	draw.DrawIslands = false

	draw.DrawSegment = func(p1 b2.Vec2, p2 b2.Vec2, color b2.HexColor) {
		x1, y1 := toScreen.Transform(gm.VecOf(float64(p1.X), float64(p1.Y))).XY()
		x2, y2 := toScreen.Transform(gm.VecOf(float64(p2.X), float64(p2.Y))).XY()

		var p vector.Path
		p.MoveTo(float32(x1), float32(y1))
		p.LineTo(float32(x2), float32(y2))

		dop := vector.DrawPathOptions{ColorScale: toColorScale(color)}
		vector.StrokePath(screen, &p, &vector.StrokeOptions{Width: 1}, &dop)
	}

	draw.DrawPolygon = func(vertices []b2.Vec2, color b2.HexColor) {
		var p vector.Path

		for _, v := range vertices {
			x, y := toScreen.Transform(gm.VecOf(float64(v.X), float64(v.Y))).XY()
			p.LineTo(float32(x), float32(y))
		}
		p.Close()

		dop := vector.DrawPathOptions{ColorScale: toColorScale(color)}
		vector.StrokePath(screen, &p, &vector.StrokeOptions{Width: 1}, &dop)
	}

	draw.DrawSolidPolygon = func(tr b2.Transform, vertices []b2.Vec2, radius float32, color b2.HexColor) {
		g := gm.IdentityAffine()
		g = g.Translate(gm.VecOf(float64(tr.P.X), float64(tr.P.Y)))
		g = g.Rotate(gm.Rad(tr.Q.Angle()))
		g = toScreen.Mul(g)

		var p vector.Path

		for _, v := range vertices {
			x, y := g.Transform(gm.VecOf(float64(v.X), float64(v.Y))).XY()
			p.LineTo(float32(x), float32(y))
		}

		p.Close()

		dop := vector.DrawPathOptions{ColorScale: toColorScale(color)}
		vector.FillPath(screen, &p, nil, &dop)
	}

	draw.DrawCircle = func(center b2.Vec2, radius float32, color b2.HexColor) {
		x, y := toScreen.Transform(gm.VecOf(float64(center.X), float64(center.Y))).XY()
		r := toScreen.Matrix.Transform(gm.VecOf(float64(radius), 0)).X

		var p vector.Path
		p.Arc(float32(x), float32(y), float32(r), 0, 2*math.Pi, vector.Clockwise)

		dop := vector.DrawPathOptions{ColorScale: toColorScale(color)}
		vector.StrokePath(screen, &p, &vector.StrokeOptions{Width: 1}, &dop)
	}

	draw.DrawSolidCircle = func(tr b2.Transform, radius float32, color b2.HexColor) {
		g := gm.IdentityAffine()
		g.Rotate(gm.Rad(tr.Q.Angle()))
		g.Translate(gm.VecOf(float64(tr.P.X), float64(tr.P.Y)))
		g = g.Mul(toScreen)

		x, y := g.Transform(gm.VecOf(float64(0), 0)).XY()
		r := g.Matrix.Transform(gm.VecOf(float64(radius), 0)).X

		var p vector.Path
		p.Arc(float32(x), float32(y), float32(r), 0, 2*math.Pi, vector.Clockwise)

		dop := vector.DrawPathOptions{ColorScale: toColorScale(color)}
		vector.FillPath(screen, &p, nil, &dop)
	}

	draw.DrawSolidCapsule = func(p1 b2.Vec2, p2 b2.Vec2, radius float32, color b2.HexColor) {
		draw.DrawSegment(p1, p2, color)
		draw.DrawCircle(p1, radius, color)
		draw.DrawCircle(p2, radius, color)
	}

	draw.DrawTransform = func(transform b2.Transform) {
		x, y := toScreen.Transform(gm.VecOf(float64(transform.P.X), float64(transform.P.Y))).XY()

		var p vector.Path
		p.MoveTo(float32(x), float32(y))
		p.LineTo(float32(x)+transform.Q.C*16, float32(x)+transform.Q.S*16)

		p.MoveTo(float32(x), float32(y))
		p.LineTo(float32(x)+-transform.Q.S*16, float32(x)+transform.Q.C*16)

		dop := vector.DrawPathOptions{ColorScale: toColorScale(0x00ff00)}
		vector.FillPath(screen, &p, nil, &dop)
	}

	draw.DrawPoint = func(c b2.Vec2, size float32, color b2.HexColor) {
		x, y := toScreen.Transform(gm.VecOf(float64(c.X), float64(c.Y))).XY()

		var p vector.Path
		p.Arc(float32(x), float32(y), size, 0, 2*math.Pi, vector.Clockwise)

		dop := vector.DrawPathOptions{ColorScale: toColorScale(color)}
		vector.FillPath(screen, &p, nil, &dop)
	}

	draw.DrawString = func(p b2.Vec2, s string, color b2.HexColor) {
		x, y := toScreen.Transform(gm.VecOf(float64(p.X), float64(p.Y))).XY()
		ebitenutil.DebugPrintAt(screen, s, int(x), int(y))
	}

	world.Draw(draw)
}

func toColorScale(h b2.HexColor) ebiten.ColorScale {
	r := float32((h>>16)&0xff) / 255.0
	g := float32((h>>8)&0xff) / 255.0
	b := float32((h>>0)&0xff) / 255.0

	var c ebiten.ColorScale
	c.Scale(r, g, b, 1)
	return c
}
