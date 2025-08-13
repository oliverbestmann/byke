package physics

import (
	"github.com/jakecoffman/cp/v2"
	"github.com/oliverbestmann/byke/gm"
)

type ToShape interface {
	MakeShape(body *cp.Body) *cp.Shape
}

type CircleShape struct {
	Radius float64
}

func (s CircleShape) MakeShape(body *cp.Body) *cp.Shape {
	return cp.NewCircle(body, s.Radius, cp.Vector{})
}

type SegmentShape struct {
	A, B   gm.Vec
	Radius float64
}

func (s SegmentShape) MakeShape(body *cp.Body) *cp.Shape {
	return cp.NewSegment(body, cp.Vector(s.A), cp.Vector(s.B), s.Radius)
}

type PolygonShape struct {
	Points []gm.Vec
	Radius float64
}

func (s PolygonShape) MakeShape(body *cp.Body) *cp.Shape {
	vertices := make([]cp.Vector, len(s.Points))
	for idx := range s.Points {
		vertices[idx] = cp.Vector(s.Points[idx])
	}

	return cp.NewPolyShape(body, len(vertices), vertices, cp.NewTransformIdentity(), s.Radius)
}
