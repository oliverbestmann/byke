package earcut

import (
	"math"
	"slices"

	"github.com/oliverbestmann/byke/gm"
)

type Point = gm.Vec

func EarCut(polygon []Point, holes [][]Point) ([]Point, []uint32) {
	outer := linkedList(polygon, true, 0)
	outer = eliminateHoles(outer, holes, uint32(len(polygon)))

	var points []Point
	points = append(points, polygon...)
	for _, hole := range holes {
		points = append(points, hole...)
	}

	return points, earcutLinked(outer, nil, 0)
}

func earcutLinked(ear *Node, triangles []uint32, pass int) []uint32 {
	if ear == nil {
		return triangles
	}

	stop := ear

	for ear.Prev != ear.Next {
		prev, next := ear.Prev, ear.Next

		// TODO invSize
		if isEar(ear) {
			// cut off the triangle
			triangles = append(triangles, prev.Index, ear.Index, next.Index)

			removeNode(ear)

			ear = next.Next
			stop = next.Next

			continue
		}

		ear = next

		if ear == stop {

			switch pass {
			case 0:
				points := filterPoints(ear, nil)
				triangles = earcutLinked(points, triangles, 1)

			case 1:
				ear, triangles = cureLocalIntersections(filterPoints(ear, nil), triangles)
				triangles = earcutLinked(ear, triangles, 2)

			case 2:
				triangles = splitEarcut(ear, triangles)
			}

			break
		}
	}

	return triangles
}

// check whether a polygon node forms a valid ear with adjacent nodes
func isEar(ear *Node) bool {
	a := ear.Prev
	b := ear
	c := ear.Next

	// reflex, can't be an ear
	if area(a.Point, b.Point, c.Point) >= 0 {
		return false
	}

	// now make sure we don't have other points inside the potential ear
	ax := a.X
	bx := b.X
	cx := c.X
	ay := a.Y
	by := b.Y
	cy := c.Y

	// triangle bbox
	x0 := min(ax, bx, cx)
	y0 := min(ay, by, cy)
	x1 := max(ax, bx, cx)
	y1 := max(ay, by, cy)

	p := c.Next

	for p != a {
		if p.X >= x0 && p.X <= x1 && p.Y >= y0 && p.Y <= y1 &&
			pointInTriangleExceptFirst(a.Point, b.Point, c.Point, p.Point) &&
			area(p.Prev.Point, p.Point, p.Next.Point) >= 0 {

			return false
		}

		p = p.Next
	}

	return true
}

func cureLocalIntersections(start *Node, triangles []uint32) (*Node, []uint32) {
	p := start

	for {
		a := p.Prev
		b := p.Next.Next

		if !a.EqualTo(b) && intersects(a.Point, p.Point, p.Next.Point, b.Point) && locallyInside(a, b) && locallyInside(b, a) {
			triangles = append(triangles, a.Index, p.Index, b.Index)

			// remove two nodes involved
			removeNode(p)
			removeNode(p.Next)
		}

		p = p.Next

		if p == start {
			break
		}
	}

	return filterPoints(p, nil), triangles
}

func intersects(p1, q1, p2, q2 Point) bool {
	o1 := sign(area(p1, q1, p2))
	o2 := sign(area(p1, q1, q2))
	o3 := sign(area(p2, q2, p1))
	o4 := sign(area(p2, q2, q1))

	if o1 != o2 && o3 != o4 {
		return true // general case
	}

	if o1 == 0 && onSegment(p1, p2, q1) {
		return true // p1, q1 and p2 are collinear and p2 lies on p1q1
	}

	if o2 == 0 && onSegment(p1, q2, q1) {
		return true // p1, q1 and q2 are collinear and q2 lies on p1q1
	}

	if o3 == 0 && onSegment(p2, p1, q2) {
		return true // p2, q2 and p1 are collinear and p1 lies on p2q2
	}

	if o4 == 0 && onSegment(p2, q1, q2) {
		return true // p2, q2 and q1 are collinear and q1 lies on p2q2
	}

	return false
}

func onSegment(p, q, r Point) bool {
	return q.X <= max(p.X, r.X) && q.X >= min(p.X, r.X) && q.Y <= max(p.Y, r.Y) && q.Y >= min(p.Y, r.Y)
}

func sign(value float64) int {
	if value < 0 {
		return -1
	}

	if value > 0 {
		return 1
	}

	return 0
}

// try splitting polygon into two and triangulate them independently
func splitEarcut(start *Node, triangles []uint32) []uint32 {
	// look for a valid diagonal that divides the polygon into two
	a := start

	for {
		b := a.Next.Next

		for b != a.Prev {
			if a.Index != b.Index && isValidDiagonal(a, b) {
				// split the polygon in two by the diagonal
				c := splitPolygon(a, b)

				// filter co-linear points around the cuts
				a = filterPoints(a, a.Next)
				c = filterPoints(c, c.Next)

				// run earcut on each half
				triangles = earcutLinked(a, triangles, 0)
				triangles = earcutLinked(c, triangles, 0)
				return triangles
			}

			b = b.Next
		}

		a = a.Next

		if a == start {
			return triangles
		}
	}
}

func isValidDiagonal(a, b *Node) bool {
	return a.Next.Index != b.Index && a.Prev.Index != b.Index && !intersectsPolygon(a, b) && // doesn't intersect other edges
		(locallyInside(a, b) && locallyInside(b, a) && middleInside(a, b) && // locally visible
			(area(a.Prev.Point, a.Point, b.Prev.Point) != 0 || area(a.Point, b.Prev.Point, b.Point) != 0) || // does not create opposite-facing sectors
			a.EqualTo(b) && area(a.Prev.Point, a.Point, a.Next.Point) > 0 && area(b.Prev.Point, b.Point, b.Next.Point) > 0) // special zero-length case
}

// check if the middle point of a polygon diagonal is inside the polygon
func middleInside(a, b *Node) bool {
	p := a
	var inside bool

	px := (a.X + b.X) / 2
	py := (a.Y + b.Y) / 2

	for {
		if ((p.Y > py) != (p.Next.Y > py)) && p.Next.Y != p.Y &&
			(px < (p.Next.X-p.X)*(py-p.Y)/(p.Next.Y-p.Y)+p.X) {

			inside = !inside
		}

		p = p.Next
		if p == a {
			break
		}
	}

	return inside
}

// check if a polygon diagonal intersects any polygon segments
func intersectsPolygon(a, b *Node) bool {
	p := a

	for {
		if p.Index != a.Index &&
			p.Next.Index != a.Index &&
			p.Index != b.Index &&
			p.Next.Index != b.Index &&
			intersects(p.Point, p.Next.Point, a.Point, b.Point) {

			return true
		}

		p = p.Next

		if p == a {
			return false
		}
	}
}

/*// go through all polygon nodes and cure small local self-intersections
function splitEarcut(start, triangles, dim, minX, minY, invSize) {
    // look for a valid diagonal that divides the polygon into two
    let a = start;
    do {
        let b = a.next.next;
        while (b !== a.prev) {
            if (a.i !== b.i && isValidDiagonal(a, b)) {
                // split the polygon in two by the diagonal
                let c = splitPolygon(a, b);

                // filter colinear points around the cuts
                a = filterPoints(a, a.next);
                c = filterPoints(c, c.next);

                // run earcut on each half
                earcutLinked(a, triangles, dim, minX, minY, invSize, 0);
                earcutLinked(c, triangles, dim, minX, minY, invSize, 0);
                return;
            }
            b = b.next;
        }
        a = a.next;
    } while (a !== start);
}

*/

func collect(node *Node) []Point {
	var points []Point

	p := node
	for {
		points = append(points, p.Point)

		p = p.Next

		if p == node {
			return points
		}
	}
}

func eliminateHoles(outer *Node, holes [][]Point, indexOffset uint32) *Node {
	var queue []*Node

	for _, hole := range holes {
		list := linkedList(hole, false, indexOffset)
		list.Steiner = len(hole) == 1
		queue = append(queue, getLeftmost(list))

		indexOffset += uint32(len(hole))
	}

	slices.SortFunc(queue, func(a, b *Node) int {
		val := compareXYSlope(a, b)
		switch {
		case val < 0:
			return -1
		case val > 0:
			return +1
		default:
			return 0
		}
	})

	// process holes from left to right
	for _, hole := range queue {
		outer = eliminateHole(hole, outer)
	}

	return outer
}

func eliminateHole(hole, outer *Node) *Node {
	bridge := findHoleBridge(hole, outer)
	if bridge == nil {
		return outer
	}

	bridgeReverse := splitPolygon(bridge, hole)

	// filter collinear points around the cuts
	filterPoints(bridgeReverse, bridgeReverse.Next)
	return filterPoints(bridge, bridge.Next)

}

// eliminate co-linear or duplicate points
func filterPoints(start, end *Node) *Node {
	if start == nil {
		return nil
	}

	if end == nil {
		end = start
	}

	p := start

	var again bool
	for {
		again = false

		if !p.Steiner && (p.EqualTo(p.Next) || area(p.Prev.Point, p.Point, p.Next.Point) == 0) {
			removeNode(p)
			end = p.Prev
			p = p.Prev

			if p == p.Next {
				break
			}

			again = true
		} else {
			p = p.Next
		}

		if !again && p == end {
			break
		}
	}

	return end
}

func findHoleBridge(hole, outer *Node) *Node {
	p := outer
	hx, hy := hole.X, hole.Y
	qx := math.Inf(-1)

	var m *Node

	// find a segment intersected by a ray from the hole's leftmost point to the left;
	// segment's endpoint with lesser x will be potential connection point
	// unless they intersect at a vertex, then choose the vertex
	if hole.EqualTo(p) {
		return p
	}

	for {
		if hole.EqualTo(p.Next) {
			return p.Next
		}

		if hy <= p.Y && hy >= p.Next.Y && p.Next.Y != p.Y {
			x := p.X + (hy-p.Y)*(p.Next.X-p.X)/(p.Next.Y-p.Y)
			if x <= hx && x > qx {
				qx = x
				if p.X < p.Next.X {
					m = p
				} else {
					m = p.Next
				}

				if x == hx {
					// hole touches outer segment; pick leftmost endpoint
					return m
				}
			}
		}

		p = p.Next

		if p == outer {
			break
		}
	}

	if m == nil {
		return nil
	}

	// look for points inside the triangle of hole point, segment intersection and endpoint;
	// if there are no points found, we have a valid connection;
	// otherwise choose the point of the minimum angle with the ray as connection point

	stop := m
	mx, my := m.XY()
	tanMin := math.Inf(1)

	p = m

	for {
		p0 := Point{X: pick(hy < my, hx, qx), Y: hy}
		p1 := Point{X: mx, Y: my}
		p2 := Point{X: pick(hy < my, qx, hx), Y: hy}

		if hx >= p.X && p.X >= mx && hx != p.X && pointInTriangle(p0, p1, p2, p.Point) {
			tan := math.Abs(hy-p.Y) / (hx - p.Y) // tangential

			if locallyInside(p, hole) &&
				(tan < tanMin || tan == tanMin && (p.X > m.X || (p.Y == m.Y && sectorContainsSector(m, p)))) {

				m = p
				tanMin = tan
			}
		}

		p = p.Next

		if p == stop {
			break
		}
	}

	return m
}

func pick(cond bool, a, b float64) float64 {
	if cond {
		return a
	} else {
		return b
	}
}

func pointInTriangle(a, b, c, p Point) bool {
	return (c.X-p.X)*(a.Y-p.Y) >= (a.X-p.X)*(c.Y-p.Y) &&
		(a.X-p.X)*(b.Y-p.Y) >= (b.X-p.X)*(a.Y-p.Y) &&
		(b.X-p.X)*(c.Y-p.Y) >= (c.X-p.X)*(b.Y-p.Y)
}

func pointInTriangleExceptFirst(a, b, c, p Point) bool {
	return a != b && pointInTriangle(a, b, c, p)
}

func area(p, q, r Point) float64 {
	return (q.Y-p.Y)*(r.X-q.X) - (q.X-p.X)*(r.Y-q.Y)
}

// check if a polygon diagonal is locally inside the polygon
func locallyInside(a, b *Node) bool {
	v := area(a.Prev.Point, a.Point, a.Next.Point)
	if v < 0 {
		return area(a.Point, b.Point, a.Next.Point) >= 0 &&
			area(a.Point, a.Prev.Point, b.Point) >= 0
	} else {
		return area(a.Point, b.Point, a.Prev.Point) < 0 ||
			area(a.Point, a.Next.Point, b.Point) < 0
	}
}

// whether sector in vertex m contains sector in vertex p in the same coordinates
func sectorContainsSector(m, p *Node) bool {
	return area(m.Prev.Point, m.Point, p.Prev.Point) < 0 &&
		area(p.Next.Point, m.Point, m.Next.Point) < 0
}

func compareXYSlope(a, b *Node) float64 {
	result := a.X - b.X

	// when the left-most point of 2 holes meet at a vertex, sort the holes counterclockwise so that when we find
	// the bridge to the outer shell is always the point that they meet at.
	if result == 0 {
		result = a.Y - b.Y
		if result == 0 {
			aSlope := (a.Next.Y - a.Y) / (a.Next.X - a.X)
			bSlope := (b.Next.Y - b.Y) / (b.Next.X - b.X)
			result = aSlope - bSlope
		}
	}

	return result
}

func signedArea(points []Point) float64 {
	var sum float64

	for i := 0; i < len(points); i++ {
		j := (i - 1 + len(points)) % len(points)
		sum += (points[j].X - points[i].X) * (points[i].Y + points[j].Y)
	}

	return sum
}

// find the leftmost node of a polygon ring
func getLeftmost(start *Node) *Node {
	p := start
	leftmost := start

	for {
		if p.X < leftmost.X || (p.X == leftmost.X && p.Y < leftmost.Y) {
			leftmost = p
		}

		p = p.Next

		if p == start {
			return leftmost
		}
	}
}

/*

// find the leftmost node of a polygon ring
function getLeftmost(start) {
    let p = start,
        leftmost = start;
    do {
        if (p.x < leftmost.x || (p.x === leftmost.x && p.y < leftmost.y)) leftmost = p;
        p = p.next;
    } while (p !== start);

    return leftmost;
}
*/

type Node struct {
	Point
	Z float64

	Prev, Next   *Node
	PrevZ, NextZ *Node

	Index   uint32
	Steiner bool
}

func (n *Node) EqualTo(other *Node) bool {
	return n.Point == other.Point
}

func (n *Node) SetNext(next *Node) {
	n.Next = next
	next.Prev = n
}

var bufNodes []Node

func createNode[I uint32 | int](idx I, point Point) *Node {
	if len(bufNodes) == 0 {
		bufNodes = make([]Node, 1024)
	}

	node := &bufNodes[0]
	bufNodes = bufNodes[1:]

	node.Point = point
	node.Index = uint32(idx)

	return node
}

func insertNode[I uint32 | int](idx I, point Point, last *Node) *Node {
	p := createNode(idx, point)
	if last == nil {
		p.Prev = p
		p.Next = p
	} else {
		p.Next = last.Next
		p.Prev = last
		last.Next.Prev = p
		last.Next = p
	}

	return p
}

func removeNode(p *Node) {
	p.Next.Prev = p.Prev
	p.Prev.Next = p.Next

	if p.PrevZ != nil {
		p.PrevZ.NextZ = p.NextZ
	}

	if p.NextZ != nil {
		p.NextZ.PrevZ = p.PrevZ
	}
}

// link two polygon vertices with a bridge; if the vertices belong to the same ring, it splits polygon into two;
// if one belongs to the outer ring and another to a hole, it merges it into a single ring
func splitPolygon(a, b *Node) *Node {
	a2 := createNode(a.Index, a.Point)
	b2 := createNode(b.Index, b.Point)

	an := a.Next
	bp := b.Prev

	a.SetNext(b)

	bp.SetNext(b2)
	b2.SetNext(a2)
	a2.SetNext(an)

	return b2
}

func linkedList(points []Point, clockwise bool, indexOffset uint32) *Node {
	var last *Node

	isClockwise := signedArea(points) > 0

	if isClockwise == clockwise {
		for i, point := range points {
			last = insertNode(uint32(i)+indexOffset, point, last)
		}
	} else {
		for i := len(points) - 1; i >= 0; i-- {
			last = insertNode(uint32(i)+indexOffset, points[i], last)
		}
	}

	// join the last polygon if closed
	if last != nil && last.EqualTo(last.Next) {
		removeNode(last)
		last = last.Next
	}

	return last
}
