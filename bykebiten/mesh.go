package bykebiten

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
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
