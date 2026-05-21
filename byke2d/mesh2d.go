package byke2d

import (
	"math"
	"unsafe"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/earcut-go"
	"golang.org/x/mobile/exp/f32"
)

type MeshColor struct {
	byke.Component[MeshColor]
	Color
}

type Mesh2d struct {
	byke.ImmutableComponent[Mesh2d]

	// Indices into Vertices to draw triangles from.
	// Length must be a multiple of three.
	Indices []uint32

	// The vertex data
	Vertices []glm.Vec2f

	// UVs define optional per-vertex UV data.
	// Must have the same length as Vertices or be empty.
	UVs []glm.Vec2f

	// Colors define optional per-vertex color.
	// Must have the same length as Vertices or be empty.
	Colors []Color

	id uint32
}

func (*Mesh2d) RequireComponents() []byke.ErasedComponent {
	return []byke.ErasedComponent{
		NewTransform(),
		InheritVisibility,
	}
}

func (m *Mesh2d) VertexCount() int {
	return len(m.Vertices)
}

func (m *Mesh2d) TriangleCount() int {
	return len(m.Indices) / 3
}

func (m *Mesh2d) ComputeUV(compute func(point glm.Vec2f) glm.Vec2f) {
	m.UVs = m.UVs[:0]

	for idx := range m.Vertices {
		uv := compute(m.Vertices[idx])
		m.UVs = append(m.UVs, uv)
	}
}

func RegularPolygon(radius float32, sides uint) Mesh2d {
	// a regular polygon is actually just a circle
	return Circle(radius, sides)
}

func Circle(radius float32, resolution uint) Mesh2d {
	size := glm.Vec2f{radius, radius}.Scale(2.0)
	return Ellipse(size, resolution)
}

func Ellipse(size glm.Vec2f, resolution uint) Mesh2d {
	halfSize := size.Scale(0.5)

	indices := make([]uint32, 0, (resolution-2)*3)
	vertices := make([]glm.Vec2f, 0, resolution)
	uvs := make([]glm.Vec2f, 0, resolution)

	startAngle := glm.Rad(math.Pi / 2)
	step := (2 * math.Pi) / glm.Rad(resolution)

	for i := range resolution {
		theta := startAngle + glm.Rad(i)*step
		cos := f32.Cos(float32(theta))
		sin := f32.Sin(float32(theta))

		x := cos * halfSize[0]
		y := sin * halfSize[1]

		vertices = append(vertices, glm.Vec2f{x, y})

		uvs = append(uvs, glm.Vec2f{
			0.5 * (cos + 1.0),
			1.0 - 0.5*(sin+1.0),
		})
	}

	for i := uint32(1); i < uint32(resolution)-1; i++ {
		indices = append(indices, 0, i, i+1)
	}

	return Mesh2d{
		Indices:  indices,
		Vertices: vertices,
		UVs:      uvs,
	}
}

func Rectangle(size glm.Vec2f) Mesh2d {
	hw, hh := size.Scale(0.5).XY()

	vertices := [4]glm.Vec2f{
		{hw, hh},
		{-hw, hh},
		{-hw, -hh},
		{hw, -hh},
	}

	uvs := [4]glm.Vec2f{
		{1, 0},
		{0, 0},
		{0, 1},
		{1, 1},
	}

	indices := []uint32{0, 1, 2, 0, 2, 3}

	return Mesh2d{
		Indices:  indices,
		Vertices: vertices[:],
		UVs:      uvs[:],
	}
}

func ConvexPolygon(vertices []glm.Vec2f) Mesh2d {
	if len(vertices) <= 2 {
		return Mesh2d{}
	}

	// create triangles for the polygon
	indices := make([]uint32, 0, (len(vertices)-2)*3)
	for i := uint32(1); i < uint32(len(vertices)-1); i++ {
		indices = append(indices, 0, i, i+1)
	}

	return Mesh2d{
		Vertices: vertices,
		Indices:  indices,
	}
}

// Polygon creates a Mesh2d from a (possibly concave) polygon. The polygon might
// contain holes. A best effort at triangulation is performed.
func Polygon(polygon []glm.Vec2f, holes ...[]glm.Vec2f) Mesh2d {
	// glm.Vec2f is binary compatible with earcut.Point[float32], so we can
	// just cast the slice data accordingly without needing to copy the actual data
	ecPolygons := castVecsToEarcutPoints(polygon)

	var ecHoles [][]earcut.Point[float32]
	for _, hole := range holes {
		ecHoles = append(ecHoles, castVecsToEarcutPoints(hole))
	}

	points, indices := earcut.Triangulate(ecPolygons, ecHoles)

	// layout is still the same, just cast
	vertices := castEarcutPointsToVecs(points)

	return Mesh2d{
		Indices:  indices,
		Vertices: vertices,
	}
}

func castVecsToEarcutPoints(vecs []glm.Vec2f) []earcut.Point[float32] {
	data := unsafe.Pointer(unsafe.SliceData(vecs))
	points := (*earcut.Point[float32])(data)
	return unsafe.Slice(points, len(vecs))
}

func castEarcutPointsToVecs(points []earcut.Point[float32]) []glm.Vec2f {
	data := unsafe.Pointer(unsafe.SliceData(points))
	vertices := (*glm.Vec2f)(data)
	return unsafe.Slice(vertices, len(points))
}

// UnionDisjoint merges disjoint meshes into one. The meshes must not
// overlap for rendering not to break.
func UnionDisjoint(meshes ...Mesh2d) Mesh2d {
	var vertexCount, indexCount int
	for _, mesh := range meshes {
		vertexCount += len(mesh.Vertices)
		indexCount += len(mesh.Indices)
	}

	vertices := make([]glm.Vec2f, 0, vertexCount)
	indices := make([]uint32, 0, indexCount)

	for _, mesh := range meshes {
		offset := uint32(len(mesh.Vertices))
		vertices = append(vertices, mesh.Vertices...)

		for _, idx := range mesh.Indices {
			indices = append(indices, idx+offset)
		}
	}

	return Mesh2d{Vertices: vertices, Indices: indices}
}

/*
func computeMeshSizeSystem(
	query byke.Query[struct {
		_ byke.Changed[Mesh2d]

		Mesh Mesh2d
		BBox *BBox
	}],
) {
	for item := range query.Items() {
		vertices := item.Mesh.Vertices
		if len(vertices) == 0 {
			// no vertices, no size
			item.BBox.Rect = glm.Rect{}
			continue
		}

		minVec := vertexToVec(vertices[0])
		maxVec := minVec

		for idx := range vertices[1:] {
			x := float32(vertices[idx].DstX)
			minVec.X = min(minVec.X, x)
			maxVec.X = max(maxVec.X, x)

			y := float32(vertices[idx].DstY)
			minVec.Y = min(minVec.Y, y)
			maxVec.Y = max(maxVec.Y, y)
		}

		// calculate bounding box
		item.BBox.Rect = glm.RectWithPoints(minVec, maxVec)
		item.BBox.ToSourceScale = glm.VecOne
	}
}
*/
