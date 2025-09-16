package physics

import (
	b2 "github.com/oliverbestmann/box2d-go"
	"github.com/oliverbestmann/byke/gm"
)

type ToShape interface {
	MakeShape(body b2.Body, def b2.ShapeDef) b2.Shape
}

type CircleShape struct {
	Radius float64
}

func (s CircleShape) MakeShape(body b2.Body, def b2.ShapeDef) b2.Shape {
	return body.CreateCircleShape(def, b2.Circle{
		Center: b2.Vec2{},
		Radius: float32(s.Radius),
	})
}

type SegmentShape struct {
	A, B   gm.Vec
	Radius float64
}

func (s SegmentShape) MakeShape(body b2.Body, def b2.ShapeDef) b2.Shape {
	if s.Radius == 0 {
		seg := b2.Segment{
			Point1: b2VecOf(s.A),
			Point2: b2VecOf(s.B),
		}

		return body.CreateSegmentShape(def, seg)
	}

	capsule := b2.Capsule{
		Center1: b2VecOf(s.A),
		Center2: b2VecOf(s.B),
		Radius:  float32(s.Radius),
	}

	return body.CreateCapsuleShape(def, capsule)
}

type PolygonShape struct {
	Points []gm.Vec
	Radius float64
}

func (s PolygonShape) MakeShape(body b2.Body, def b2.ShapeDef) b2.Shape {
	points := make([]b2.Vec2, len(s.Points))
	for idx := range s.Points {
		points[idx] = b2VecOf(s.Points[idx])
	}

	hull, ok := b2.ComputeHull(points)
	if !ok {
		panic("invalid hull")
	}

	poly := b2.MakePolygon(hull, float32(s.Radius))
	return body.CreatePolygonShape(def, poly)
}

func b2VecOf(vec gm.Vec) b2.Vec2 {
	return b2.Vec2{
		X: float32(vec.X),
		Y: float32(vec.Y),
	}
}
