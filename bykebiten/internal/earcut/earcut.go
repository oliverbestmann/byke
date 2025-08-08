package earcut

import (
	"fmt"
	"math"
	"slices"

	"github.com/oliverbestmann/byke/gm"
)

type Point = gm.Vec

func EarCut(polygon []Point, holes [][]Point) ([]Point, []uint32) {
	// count the number of points
	pointCount := len(polygon)
	for _, hole := range holes {
		pointCount += len(hole)
	}

	// collect all points
	points := make([]Point, 0, pointCount)
	points = append(points, polygon...)
	for _, hole := range holes {
		points = append(points, hole...)
	}

	var invSize invSize
	if len(points) > 80 {
		invSize = calculateInvSize(points)
	}

	// create a linked Node list
	outer := linkedList(polygon, true, 0)

	if outer == nil || outer.Next == outer.Prev {
		return nil, nil
	}

	outer = eliminateHoles(outer, holes, uint32(len(polygon)))

	fmt.Println("OuterNode", outer.Index, "prev=", outer.Prev.Index)
	return points, earcutLinked(outer, nil, invSize, 0)
}

type invSize struct {
	Scale      float64
	MinX, MinY float64
}

func (i *invSize) Valid() bool {
	return i.Scale > 0
}

func calculateInvSize(points []Point) invSize {
	minX := points[0].X
	maxX := points[0].X

	minY := points[0].Y
	maxY := points[0].Y

	for _, point := range points[1:] {
		minX = min(minX, point.X)
		minY = min(minY, point.Y)
		maxX = max(maxX, point.X)
		maxY = max(maxY, point.Y)
	}

	invSize := invSize{
		MinX: minX,
		MinY: minY,
	}

	maxSize := max(maxX-minX, maxY-minY)
	if maxSize != 0 {
		invSize.Scale = 32767 / maxSize
	}

	return invSize
}

func earcutLinked(ear *Node, triangles []uint32, invSize invSize, pass int) []uint32 {
	if ear == nil {
		return triangles
	}

	if pass == 0 && invSize.Valid() {
		indexCurve(ear, invSize)
	}

	stop := ear

	for ear.Prev != ear.Next {
		prev, next := ear.Prev, ear.Next

		var val bool
		if invSize.Valid() {
			val = isEarHashed(ear, invSize)
		} else {
			val = isEar(ear)
		}

		if val {
			// cut off the triangle
			triangles = append(triangles, prev.Index, ear.Index, next.Index)

			fmt.Println("RemoveEar", ear.Index)
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
				triangles = earcutLinked(points, triangles, invSize, 1)

			case 1:
				// if this didn't work, try curing all small self-intersections locally
				ear, triangles = cureLocalIntersections(filterPoints(ear, nil), triangles)
				triangles = earcutLinked(ear, triangles, invSize, 2)

			case 2:
				triangles = splitEarcut(ear, triangles, invSize)
			}

			break
		}
	}

	return triangles
}

func indexCurve(start *Node, invSize invSize) {
	p := start

	for {
		if p.Z == 0 {
			p.Z = zOrder(p.Point, invSize)
		}

		p.PrevZ = p.Prev
		p.NextZ = p.Next

		p = p.Next

		if p == start {
			break
		}
	}

	p.PrevZ.NextZ = nil
	p.PrevZ = nil

	sortLinked(p)
}

// z-order of a point given coords and inverse of the longer side of data bbox
func zOrder(point Point, invSize invSize) uint32 {
	xf, yf := point.XY()

	// coords are transformed into non-negative 15-bit integer range
	x := int32((xf - invSize.MinX) * invSize.Scale)
	y := int32((yf - invSize.MinY) * invSize.Scale)

	x = (x | (x << 8)) & 0x00ff00ff
	x = (x | (x << 4)) & 0x0f0f0f0f
	x = (x | (x << 2)) & 0x33333333
	x = (x | (x << 1)) & 0x55555555

	y = (y | (y << 8)) & 0x00ff00ff
	y = (y | (y << 4)) & 0x0f0f0f0f
	y = (y | (y << 2)) & 0x33333333
	y = (y | (y << 1)) & 0x55555555

	return uint32(x | (y << 1))
}

// Simon Tatham's linked list merge sort algorithm
// http://www.chiark.greenend.org.uk/~sgtatham/algorithms/listsort.html
func sortLinked(list *Node) *Node {
	inSize := 1

	for {
		var e, tail *Node
		var numMerges int

		p := list
		list = nil

		for p != nil {
			numMerges += 1
			q := p
			pSize := 0

			for range inSize {
				pSize++
				q = q.NextZ
				if q == nil {
					break
				}
			}

			qSize := inSize

			for pSize > 0 || (qSize > 0 && q != nil) {
				if pSize != 0 && (qSize == 0 || q == nil || p.Z <= q.Z) {
					e = p
					p = p.NextZ
					pSize--
				} else {
					e = q
					q = q.NextZ
					qSize--
				}

				if tail != nil {
					tail.NextZ = e
				} else {
					list = e
				}

				e.PrevZ = tail
				tail = e
			}

			p = q
		}

		tail.NextZ = nil
		inSize *= 2

		if numMerges <= 1 {
			break
		}
	}

	return list
}

// check whether a polygon node forms a valid ear with adjacent nodes
func isEar(ear *Node) bool {
	// fmt.Println("Ear", ear.Index)

	a := ear.Prev
	b := ear
	c := ear.Next

	// reflex, can't be an ear
	if area(a.Point, b.Point, c.Point) >= 0 {
		return false
	}

	// now make sure we don't have other points inside the potential ear
	// triangle bbox
	x0 := min(a.X, b.X, c.X)
	y0 := min(a.Y, b.Y, c.Y)
	x1 := max(a.X, b.X, c.X)
	y1 := max(a.Y, b.Y, c.Y)

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

func isEarHashed(ear *Node, invSize invSize) bool {
	a := ear.Prev
	b := ear
	c := ear.Next

	if area(a.Point, b.Point, c.Point) >= 0 {
		// reflex, can't be an ear
		return false
	}

	// triangle bbox
	x0 := min(a.X, b.X, c.X)
	y0 := min(a.Y, b.Y, c.Y)
	x1 := max(a.X, b.X, c.X)
	y1 := max(a.Y, b.Y, c.Y)

	// z-order range for the current triangle bbox;
	minZ := zOrder(Point{X: x0, Y: y0}, invSize)
	maxZ := zOrder(Point{X: x1, Y: y1}, invSize)

	p := ear.PrevZ
	n := ear.NextZ

	// look for points inside the triangle in both directions
	for p != nil && p.Z >= minZ && n != nil && n.Z <= maxZ {
		if p.X >= x0 &&
			p.X <= x1 &&
			p.Y >= y0 &&
			p.Y <= y1 &&
			p != a &&
			p != c &&
			pointInTriangleExceptFirst(a.Point, b.Point, c.Point, p.Point) &&
			area(p.Prev.Point, p.Point, p.Next.Point) >= 0 {
			return false
		}

		p = p.PrevZ

		if n.X >= x0 &&
			n.X <= x1 &&
			n.Y >= y0 &&
			n.Y <= y1 &&
			n != a &&
			n != c &&
			pointInTriangleExceptFirst(a.Point, b.Point, c.Point, n.Point) &&
			area(n.Prev.Point, n.Point, n.Next.Point) >= 0 {
			return false
		}

		n = n.NextZ
	}

	// look for remaining points in decreasing z-order
	for p != nil && p.Z >= minZ {
		if p.X >= x0 &&
			p.X <= x1 &&
			p.Y >= y0 &&
			p.Y <= y1 &&
			p != a &&
			p != c &&
			pointInTriangleExceptFirst(a.Point, b.Point, c.Point, p.Point) &&
			area(p.Prev.Point, p.Point, p.Next.Point) >= 0 {
			return false
		}

		p = p.PrevZ
	}

	// look for remaining points in increasing z-order
	for n != nil && n.Z <= maxZ {
		if n.X >= x0 &&
			n.X <= x1 &&
			n.Y >= y0 &&
			n.Y <= y1 &&
			n != a &&
			n != c &&
			pointInTriangleExceptFirst(a.Point, b.Point, c.Point, n.Point) &&
			area(n.Prev.Point, n.Point, n.Next.Point) >= 0 {
			return false
		}

		n = n.NextZ
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

			p = b
			start = b
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
	if value > 0 {
		return 1
	}

	if value < 0 {
		return -1
	}

	return 0
}

// try splitting polygon into two and triangulate them independently
func splitEarcut(start *Node, triangles []uint32, invSize invSize) []uint32 {
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
				triangles = earcutLinked(a, triangles, invSize, 0)
				triangles = earcutLinked(c, triangles, invSize, 0)
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

func eliminateHoles(outer *Node, holes [][]Point, indexOffset uint32) *Node {
	var queue []*Node

	for _, hole := range holes {
		list := linkedList(hole, false, indexOffset)
		list.Steiner = len(hole) == 1
		queue = append(queue, getLeftmost(list))

		indexOffset += uint32(len(hole))
	}

	slices.SortFunc(queue, func(a, b *Node) int {
		return sign(compareXYSlope(a, b))
	})

	// process holes from left to right
	for _, hole := range queue {
		fmt.Println("Queue Hole", hole.Index)
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

	for {
		var again bool

		if !p.Steiner && (p.EqualTo(p.Next) || clampZero(area(p.Prev.Point, p.Point, p.Next.Point)) == 0) {
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
				m = pick(p.X < p.Next.X, p, p.Next)

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
			tan := math.Abs(hy-p.Y) / (hx - p.X) // tangential

			if locallyInside(p, hole) &&
				(tan < tanMin || (tan == tanMin && (p.X > m.X || (p.X == m.X && sectorContainsSector(m, p))))) {

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

func pick[T any](cond bool, a, b T) T {
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
	return a != p && pointInTriangle(a, b, c, p)
}

// signed area of a triangle
func area(p, q, r Point) float64 {
	return (q.Y-p.Y)*(r.X-q.X) - (q.X-p.X)*(r.Y-q.Y)
}

func clampZero(value float64) float64 {
	if math.Abs(value) < 1e-9 {
		// needed for issue142, maybe different floating point handling?
		value = 0
	}

	return value
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

type Node struct {
	Point
	Z uint32

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
	fmt.Println("Split at", a.Index, b.Index)
	a2 := createNode(a.Index, a.Point)
	b2 := createNode(b.Index, b.Point)

	an := a.Next
	bp := b.Prev

	a.SetNext(b)

	a2.SetNext(an)
	b2.SetNext(a2)
	bp.SetNext(b2)

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
