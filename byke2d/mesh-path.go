package byke2d

import (
	"math"

	"github.com/oliverbestmann/byke/byke2d/glm"
)

type operationType uint32

const (
	opMove       operationType = 1
	opLine       operationType = 2
	opQuadCurve  operationType = 3
	opCubicCurve operationType = 4
	opClose      operationType = 5
)

type pathOp struct {
	Type    operationType
	End     glm.Vec2f
	Control [2]glm.Vec2f
}

// Path can be used to construct a mesh from lines and curves.
type Path struct {
	ops    []pathOp
	closed bool
}

func (p *Path) MoveTo(pos glm.Vec2f) {
	p.ops = append(p.ops, pathOp{
		Type: opMove,
		End:  pos,
	})
}

func (p *Path) LineTo(pos glm.Vec2f) {
	p.ops = append(p.ops, pathOp{
		Type: opLine,
		End:  pos,
	})
}

func (p *Path) Close() {
	p.ops = append(p.ops, pathOp{
		Type: opClose,
	})
}

func (p *Path) QuadCurveTo(control, end glm.Vec2f) {
	p.ops = append(p.ops, pathOp{
		Type:    opQuadCurve,
		End:     end,
		Control: [2]glm.Vec2f{control},
	})
}

func (p *Path) CubicCurveTo(control1, control2, end glm.Vec2f) {
	p.ops = append(p.ops, pathOp{
		Type:    opCubicCurve,
		End:     end,
		Control: [2]glm.Vec2f{control1, control2},
	})
}

func (p *Path) ToMesh(unitScale float32) *Mesh {
	contour := p.ToContour(unitScale)
	return Polygon(contour)
}

func (p *Path) ToContour(unitScale float32) []glm.Vec2f {
	var points []glm.Vec2f

	var curr glm.Vec2f

	for _, op := range p.ops {
		switch op.Type {
		case opMove:
			points = append(points, op.End)

		case opLine:
			points = append(points, op.End)

		case opQuadCurve:
			// const segments = 16
			// points = appendQuadCurve(points, curr, op.Control[0], op.TryEnd, segments)
			adaptiveQuadCurve(curr, op.Control[0], op.End, unitScale, &points)

		case opCubicCurve:
			// const segments = 16
			// points = appendCubicCurve(points, curr, op.Control[0], op.Control[1], op.TryEnd, segments)
			adaptiveCubicCurve(curr, op.Control[0], op.Control[1], op.End, unitScale, &points)

		case opClose:
			if len(points) > 0 {
				points = append(points, points[0])
			}
		}

		curr = op.End
	}

	if len(points) > 0 {
		// cleanup duplicate points
		pointsClean := points[:1]

		prev := points[0]
		for _, point := range points[1:] {
			if prev == point {
				// skip this point
				continue
			}

			// distinct point, keep this one
			pointsClean = append(pointsClean, point)
			prev = point
		}

		points = pointsClean
	}

	return points
}

// sampleQuadCurve computes a point on a quadratic Bézier at parameter t in [0,1].
func sampleQuadCurve(p0, p1, p2 glm.Vec2f, t float32) glm.Vec2f {
	// B(t) = (1-t)^2 * p0 + 2(1-t)t * p1 + t^2 * p2
	omt := 1.0 - t
	omt2 := omt * omt
	t2 := t * t

	x := omt2*p0[0] + 2*omt*t*p1[0] + t2*p2[0]
	y := omt2*p0[1] + 2*omt*t*p1[1] + t2*p2[1]
	return glm.Vec2f{x, y}
}

func sampleCubicCurve(p0, p1, p2, p3 glm.Vec2f, t float32) glm.Vec2f {
	// B(t) = (1-t)^3 * p0 +
	//        3(1-t)^2 t * p1 +
	//        3(1-t) t^2 * p2 +
	//        t^3 * p3
	omt := 1.0 - t
	omt2 := omt * omt
	omt3 := omt2 * omt
	t2 := t * t
	t3 := t2 * t

	x := omt3*p0[0] +
		3*omt2*t*p1[0] +
		3*omt*t2*p2[0] +
		t3*p3[0]

	y := omt3*p0[1] +
		3*omt2*t*p1[1] +
		3*omt*t2*p2[1] +
		t3*p3[1]

	return glm.Vec2f{x, y}
}

func appendQuadCurve(points []glm.Vec2f, start, control, end glm.Vec2f, segments int) []glm.Vec2f {
	segments = max(1, segments)

	for i := 0; i <= segments; i++ {
		t := float32(i) / float32(segments)
		points = append(points, sampleQuadCurve(start, control, end, t))
	}

	return points
}

func appendCubicCurve(points []glm.Vec2f, start, c1, c2, end glm.Vec2f, segments int) []glm.Vec2f {
	segments = max(1, segments)

	for i := 0; i <= segments; i++ {
		t := float32(i) / float32(segments)
		points = append(points, sampleCubicCurve(start, c1, c2, end, t))
	}

	return points
}

func adaptiveQuadCurve(p0, p1, p2 glm.Vec2f, flatness float32, out *[]glm.Vec2f) {
	if quadFlatEnough(p0, p1, p2, flatness) {
		*out = append(*out, p0, p2)
		return
	}

	q0 := mid(p0, p1)
	q1 := mid(p1, p2)
	m := mid(q0, q1)

	adaptiveQuadCurve(p0, q0, m, flatness, out)
	adaptiveQuadCurve(m, q1, p2, flatness, out)
}

func adaptiveCubicCurve(p0, p1, p2, p3 glm.Vec2f, flatness float32, out *[]glm.Vec2f) {
	if cubicFlatEnough(p0, p1, p2, p3, flatness) {
		// Emit segment endpoints
		*out = append(*out, p0, p3)
		return
	}

	q0 := mid(p0, p1)
	q1 := mid(p1, p2)
	q2 := mid(p2, p3)

	r0 := mid(q0, q1)
	r1 := mid(q1, q2)

	s := mid(r0, r1)

	adaptiveCubicCurve(p0, q0, r0, s, flatness, out)
	adaptiveCubicCurve(s, r1, q2, p3, flatness, out)
}

func quadFlatEnough(p0, p1, p2 glm.Vec2f, threshold float32) bool {
	return pointLineDistance(p1, p0, p2) <= threshold
}

func cubicFlatEnough(p0, p1, p2, p3 glm.Vec2f, threshold float32) bool {
	// Distance of p1 and p2 from the line p0-p3
	d1 := pointLineDistance(p1, p0, p3)
	d2 := pointLineDistance(p2, p0, p3)
	return d1 <= threshold && d2 <= threshold
}

func mid(a, b glm.Vec2f) glm.Vec2f {
	return glm.Vec2f{(a[0] + b[0]) * 0.5, (a[1] + b[1]) * 0.5}
}

func pointLineDistance(p, a, b glm.Vec2f) float32 {
	ab := glm.Vec2f{b[0] - a[0], b[1] - a[1]}
	ap := glm.Vec2f{p[0] - a[0], p[1] - a[1]}

	// Project AP onto AB
	t := (ap[0]*ab[0] + ap[1]*ab[1]) / (ab[0]*ab[0] + ab[1]*ab[1])

	// Closest point on AB
	closest := glm.Vec2f{a[0] + t*ab[0], a[1] + t*ab[1]}

	// Distance AP→closest
	dx := p[0] - closest[0]
	dy := p[1] - closest[1]
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}
