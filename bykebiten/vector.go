package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/internal/arch"
	"math"
)

type Direction = vector.Direction

const Clockwise = vector.Clockwise
const CounterClockwise = vector.CounterClockwise

type LineCap = vector.LineCap

const (
	LineCapButt   = vector.LineCapButt
	LineCapRound  = vector.LineCapRound
	LineCapSquare = vector.LineCapSquare
)

type LineJoin = vector.LineJoin

const (
	LineJoinMiter = vector.LineJoinMiter
	LineJoinBevel = vector.LineJoinBevel
	LineJoinRound = vector.LineJoinRound
)

var _ = byke.ValidateComponent[Fill]()
var _ = byke.ValidateComponent[Stroke]()

type Fill struct {
	byke.ComparableComponent[Fill]
	Color     color.Color
	Antialias bool
}

type Stroke struct {
	byke.ComparableComponent[Stroke]
	Color color.Color

	// Width is the stroke width in pixels.
	Width float64

	// MiterLimit is the miter limit for LineJoinMiter.
	// For details, see https://developer.mozilla.org/en-US/docs/Web/SVG/Attribute/stroke-miterlimit.
	MiterLimit float64

	// LineCap is the way in which how the ends of the stroke are rendered.
	// Line caps are not rendered when the subpath is marked as closed.
	//
	// The default (zero) value is LineCapButt.
	LineCap LineCap

	// LineJoin is the way in which how two segments are joined.
	//
	// The default (zero) value is LineJoinMiter.
	LineJoin LineJoin

	// Enable antialiasing during rendering
	Antialias bool
}

type Path struct {
	byke.ComparableComponent[Path]

	inner_ *vector.Path

	// the inner_ vector.Path is not hashable. To still make this type comparable, we
	// use a pointer to the actual path and then update the version field each time
	// the inner_ path is modified to change the components hash.
	version uint64
}

func (*Path) RequireComponents() []arch.ErasedComponent {
	components := []arch.ErasedComponent(nil)
	return append(components, commonRenderComponents...)
}

func (p *Path) inner() *vector.Path {
	if p.inner_ == nil {
		p.inner_ = &vector.Path{}
	}

	return p.inner_
}

func (p *Path) Rectangle(rect gm.Rect) {
	p.MoveTo(rect.TopLeft())
	p.LineTo(rect.TopRight())
	p.LineTo(rect.BottomRight())
	p.LineTo(rect.BottomLeft())
	p.Close()
}

func (p *Path) Circle(center gm.Vec, radius float64) {
	p.MoveTo(center.Add(gm.Vec{X: radius}))
	p.Arc(center, radius, 0, 2*math.Pi, Clockwise)
}

func (p *Path) LineTo(vec gm.Vec) {
	p.version += 1
	p.inner().LineTo(float32(vec.X), float32(vec.Y))
}

func (p *Path) MoveTo(vec gm.Vec) {
	p.version += 1
	p.inner().MoveTo(float32(vec.X), float32(vec.Y))
}

func (p *Path) QuadTo(control, dest gm.Vec) {
	p.version += 1
	p.inner().QuadTo(float32(control.X), float32(control.Y), float32(dest.X), float32(dest.Y))
}

func (p *Path) CubicTo(firstControl, secondControl, dest gm.Vec) {
	p.version += 1
	p.inner().CubicTo(float32(firstControl.X), float32(firstControl.Y), float32(secondControl.X), float32(secondControl.Y), float32(dest.X), float32(dest.Y))
}

func (p *Path) Arc(center gm.Vec, radius float64, startAngle, endAngle gm.Rad, direction Direction) {
	p.version += 1
	p.inner().Arc(
		float32(center.X), float32(center.Y),
		float32(radius), float32(startAngle), float32(endAngle),
		direction,
	)
}

func (p *Path) ArcTo(firstControl, secondControl gm.Vec, radius float64) {
	p.version += 1
	p.inner().ArcTo(
		float32(firstControl.X), float32(firstControl.Y),
		float32(secondControl.X), float32(secondControl.Y),
		float32(radius),
	)
}

func (p *Path) Close() {
	p.version += 1
	p.inner().Close()
}

func computeCachedVertices(
	query byke.Query[struct {
		byke.Or[byke.Changed[Path], byke.Changed[Stroke]]

		Path   Path
		Anchor Anchor

		BBox *BBox
	}],
) {

	for item := range query.Items() {
		// get the actual path
		path := item.Path.inner_
		if path == nil {
			continue
		}

		bbox := path.Bounds()

		minVec := gm.VecOf(float64(bbox.Min.X), float64(bbox.Min.Y))
		maxVec := gm.VecOf(float64(bbox.Max.X), float64(bbox.Max.Y))

		// calculate bounding box
		size := maxVec.Sub(minVec)
		origin := item.Anchor.MulEach(size).Mul(-1)
		item.BBox.Rect = gm.RectWithOriginAndSize(origin, size)
		item.BBox.ToSourceScale = gm.VecOne
		item.BBox.LocalOrigin = minVec
	}
}
