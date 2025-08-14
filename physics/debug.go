package physics

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
)

func debugSystem(
	space cpSpace,
	renderTarget bykebiten.DefaultRenderTarget,
	screenSize bykebiten.ScreenSize,
	cameraQuery byke.Query[struct {
		Projection bykebiten.OrthographicProjection
		Transform  bykebiten.GlobalTransform
	}],
) {
	item, _ := cameraQuery.Single()

	tr := bykebiten.CalculateWorldToScreenTransform(item.Projection, item.Transform, screenSize.Vec)

	cp.DrawSpace(space.Space, debugImage{Image: renderTarget.Image, Transform: tr})
}

type debugImage struct {
	Image     *ebiten.Image
	Transform gm.Affine
}

func (d debugImage) draw(p vector.Path, outline cp.FColor, fill cp.FColor) {
	dpo := &vector.DrawPathOptions{}
	dpo.ColorScale.Scale(fill.R*fill.A, fill.G*fill.A, fill.B*fill.A, fill.A)
	vector.FillPath(d.Image, &p, &vector.FillOptions{}, dpo)

	*dpo = vector.DrawPathOptions{}
	dpo.ColorScale.Scale(outline.R*outline.A, outline.G*outline.A, outline.B*outline.A, outline.A)
	vector.StrokePath(d.Image, &p, &vector.StrokeOptions{Width: 1}, dpo)

}

func (d debugImage) DrawCircle(pos cp.Vector, angle, radius float64, outline, fill cp.FColor, data interface{}) {
	tpos := d.Transform.Transform(gm.Vec(pos))
	radius = d.Transform.TransformVec(gm.Vec{X: radius}).Length()

	var p vector.Path
	p.Arc(float32(tpos.X), float32(tpos.Y), float32(radius), 0, math.Pi*2, vector.Clockwise)
	p.MoveTo(float32(tpos.X), float32(tpos.Y))
	p.LineTo(float32(math.Cos(angle)*10), float32(-math.Sin(angle)*10))

	d.draw(p, outline, fill)
}

func (d debugImage) DrawSegment(a, b cp.Vector, fill cp.FColor, data interface{}) {
	ta := d.Transform.Transform(gm.Vec(a))
	tb := d.Transform.Transform(gm.Vec(b))

	var p vector.Path
	p.LineTo(float32(ta.X), float32(ta.Y))
	p.LineTo(float32(tb.X), float32(tb.Y))
	d.draw(p, fill, cp.FColor{})
}

func (d debugImage) DrawFatSegment(a, b cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	if radius < 0.01 {
		d.DrawSegment(a, b, outline, data)
		return
	}

	r := gm.Vec(a).Sub(gm.Vec(b)).Normalized().Rotated(gm.DegToRad(90)).Mul(radius)
	r = d.Transform.TransformVec(r)

	radius = r.Length()

	ta := d.Transform.Transform(gm.Vec(a))
	tb := d.Transform.Transform(gm.Vec(b))

	c0 := ta.Sub(r)
	c1 := tb.Sub(r)
	c2 := tb.Add(r)
	c3 := ta.Add(r)

	var p vector.Path
	p.MoveTo(float32(c0.X), float32(c0.Y))
	p.LineTo(float32(c1.X), float32(c1.Y))
	p.Arc(float32(tb.X), float32(tb.Y), float32(radius), float32(r.Angle()), float32(r.Angle()+math.Pi), vector.Clockwise)
	p.LineTo(float32(c2.X), float32(c2.Y))
	p.LineTo(float32(c3.X), float32(c3.Y))
	p.Arc(float32(ta.X), float32(ta.Y), float32(radius), float32(r.Angle()), float32(r.Angle()+math.Pi), vector.CounterClockwise)

	d.draw(p, outline, fill)
}

func (d debugImage) DrawPolygon(count int, verts []cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	for idx := range count - 1 {
		a, b := verts[idx], verts[idx+1]
		d.DrawFatSegment(a, b, radius, outline, fill, data)
	}
}

func (d debugImage) DrawDot(size float64, pos cp.Vector, fill cp.FColor, data interface{}) {
	d.DrawCircle(pos, 0, size/2, fill, fill, data)
}

func (d debugImage) Flags() uint {
	return 0
}

func (d debugImage) OutlineColor() cp.FColor {
	return cp.FColor{G: 1, A: 1}
}

func (d debugImage) ShapeColor(shape *cp.Shape, data interface{}) cp.FColor {
	return cp.FColor{G: 1, A: 0.5}
}

func (d debugImage) ConstraintColor() cp.FColor {
	return cp.FColor{R: 1, G: 0.75, A: 1}
}

func (d debugImage) CollisionPointColor() cp.FColor {
	return cp.FColor{R: 1, A: 1}
}

func (d debugImage) Data() interface{} {
	return nil
}
