package bykebiten

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/tchayen/triangolatte"
)

type Vertex = ebiten.Vertex

type Mesh struct {
	byke.ImmutableComponent[Mesh]
	Vertices []Vertex
	Indices  []uint32
}

func (Mesh) RequireComponents() []spoke.ErasedComponent {
	components := []spoke.ErasedComponent{
		Blend{},
		Filter{},
	}

	return append(components, commonRenderComponents...)
}

func RegularPolygon(radius float64, sides uint) Mesh {
	// a regular polygon is actually just a circle
	return Circle(radius, sides)
}

func Circle(radius float64, resolution uint) Mesh {
	return Ellipse(gm.VecSplat(radius).Mul(2.0), resolution)
}

func Ellipse(size gm.Vec, resolution uint) Mesh {
	halfSize := size.Mul(0.5)

	indices := make([]uint32, 0, (resolution-2)*3)
	vertices := make([]Vertex, 0, resolution)

	startAngle := gm.Rad(math.Pi / 2)
	step := (2 * math.Pi) / gm.Rad(resolution)

	for i := range resolution {
		theta := startAngle + gm.Rad(i)*step
		sin, cos := theta.SinCos()

		x := cos * halfSize.X
		y := sin * halfSize.Y

		vertices = append(vertices, Vertex{
			DstX:    float32(x),
			DstY:    float32(y),
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: float32(0.5 * (cos + 1.0)),
			Custom1: float32(1.0 - 0.5*(sin+1.0)),
		})
	}

	for i := uint32(1); i < uint32(resolution)-1; i++ {
		indices = append(indices, 0, i, i+1)
	}

	return Mesh{
		Vertices: vertices,
		Indices:  indices,
	}
}

func Rectangle(size gm.Vec) Mesh {
	hw, hh := size.Mul(0.5).XY()

	vertices := []ebiten.Vertex{
		{
			DstX:    float32(hw),
			DstY:    float32(hh),
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: 1.0,
			Custom1: 0.0,
		},
		{
			DstX:    float32(-hw),
			DstY:    float32(hh),
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: 0.0,
			Custom1: 0.0,
		},
		{
			DstX:    float32(-hw),
			DstY:    float32(-hh),
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: 0.0,
			Custom1: 1.0,
		},
		{
			DstX:    float32(hw),
			DstY:    float32(-hh),
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: 1.0,
			Custom1: 1.0,
		},
	}

	indices := []uint32{0, 1, 2, 0, 2, 3}

	return Mesh{
		Vertices: vertices,
		Indices:  indices,
	}
}

func ConvexPolygon(points []gm.Vec) Mesh {
	if len(points) <= 2 {
		return Mesh{}
	}

	indices := make([]uint32, 0, (len(points)-2)*3)
	vertices := make([]Vertex, 0, len(points))

	for _, point := range points {
		vertices = append(vertices, ebiten.Vertex{
			DstX:   float32(point.X),
			DstY:   float32(point.Y),
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		})
	}

	for i := uint32(1); i < uint32(len(points)-1); i++ {
		indices = append(indices, 0, i, i+1)
	}

	return Mesh{
		Vertices: vertices,
		Indices:  indices,
	}
}

func Polygon(polygon []gm.Vec, holes ...[]gm.Vec) Mesh {
	pointsOf := func(vecs []gm.Vec) []triangolatte.Point {
		data := unsafe.SliceData(vecs)
		return unsafe.Slice((*triangolatte.Point)(data), len(vecs))
	}

	points := pointsOf(polygon)

	if len(holes) > 0 {
		polygons := [][]triangolatte.Point{points}

		for _, hole := range holes {
			polygons = append(polygons, pointsOf(hole))
		}

		joined, err := triangolatte.JoinHoles(polygons)
		if err != nil {
			panic(fmt.Errorf("joining holes: %w", err))
		}

		points = joined
	}

	triangles, err := triangolatte.Polygon(points)
	if err != nil {
		panic(fmt.Errorf("triangulate: %w", err))
	}

	vertices := make([]ebiten.Vertex, 0, len(triangles)/6)

	for idx := 0; idx < len(triangles); idx += 6 {
		vertices = append(vertices,
			ebiten.Vertex{
				DstX:   float32(triangles[idx+0]),
				DstY:   float32(triangles[idx+1]),
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			ebiten.Vertex{
				DstX:   float32(triangles[idx+2]),
				DstY:   float32(triangles[idx+3]),
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			ebiten.Vertex{
				DstX:   float32(triangles[idx+4]),
				DstY:   float32(triangles[idx+5]),
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
		)
	}

	indices := make([]uint32, len(vertices))
	for idx := range len(vertices) {
		indices[idx] = uint32(idx)
	}

	return Mesh{
		Vertices: vertices,
		Indices:  indices,
	}
}

func computeMeshSizeSystem(
	query byke.Query[struct {
		_ byke.Changed[Mesh]

		Mesh Mesh
		BBox *BBox
	}],
) {
	for item := range query.Items() {
		vertices := item.Mesh.Vertices
		if len(vertices) == 0 {
			// no vertices, no size
			item.BBox.Rect = gm.Rect{}
			continue
		}

		minVec := vertexToVec(vertices[0])
		maxVec := minVec

		for idx := range vertices[1:] {
			x := float64(vertices[idx].DstX)
			minVec.X = min(minVec.X, x)
			maxVec.X = max(maxVec.X, x)

			y := float64(vertices[idx].DstY)
			minVec.Y = min(minVec.Y, y)
			maxVec.Y = max(maxVec.Y, y)
		}

		// calculate bounding box
		item.BBox.Rect = gm.RectWithPoints(minVec, maxVec)
		item.BBox.ToSourceScale = gm.VecOne
	}
}

func vertexToVec(vertex Vertex) gm.Vec {
	return gm.Vec{X: float64(vertex.DstX), Y: float64(vertex.DstY)}
}
