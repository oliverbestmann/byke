package byke2d

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/earcut-go"
	"github.com/oliverbestmann/mikktspace-go"
	"github.com/oliverbestmann/webgpu/wgpu"
	"golang.org/x/mobile/exp/f32"
)

type Mesh struct {
	// indices indexes into vertices to draw triangles from.
	// Length of indices must be a multiple of three.
	indices []uint32

	// vertices is the vertex position data
	vertices []glm.Vec3f

	// Additional vertex attributes
	attributes VertexAttributes

	// a mesh can contain multiple morph targets
	morphTargets [][]MorphAttributes

	// set to true if the mesh is uploaded to the gpu
	version uint32
}

func MeshOf(indices []uint32, vertices []glm.Vec3f) *Mesh {
	return &Mesh{
		indices:  indices,
		vertices: vertices,
	}
}

func (m *Mesh) WithVertices(vertices []glm.Vec3f) *Mesh {
	m.version += 1
	m.vertices = vertices
	return m
}

func (m *Mesh) WithAttributes(attr VertexAttribute, values []byte) *Mesh {
	m.version += 1
	m.attributes.Insert(attr, values)
	return m
}

// WithMorphTarget adds another morph target to this vertex
func (m *Mesh) WithMorphTarget(target []MorphAttributes) *Mesh {
	if len(target) != len(m.vertices) {
		panic(fmt.Errorf("got %d morph attributes for %d vertices", len(target), len(m.vertices)))
	}

	m.version += 1
	m.morphTargets = append(m.morphTargets, target)
	return m
}

// Vertices returns the vertices of this Mesh. You should not modify the
// returned slice.
func (m *Mesh) Vertices() []glm.Vec3f {
	return m.vertices
}

func (m *Mesh) ComputeUV(compute func(point glm.Vec3f) glm.Vec2f) {
	var uvs []glm.Vec2f

	for _, vertex := range m.vertices {
		uv := compute(vertex)
		uvs = append(uvs, uv)
	}

	m.WithAttributes(VertexAttributeUV, wgpu.ToBytes(uvs))
}

func (m *Mesh) VertexCount() int {
	return len(m.vertices)
}

func (m *Mesh) MorphTargetCount() int {
	return len(m.morphTargets)
}

// Transform applies the given matrix to all vertices within this mesh
func (m *Mesh) Transform(tr glm.Mat4f) {
	for idx := range m.vertices {
		m.vertices[idx] = tr.Transform3(m.vertices[idx])
	}
}

func (m *Mesh) AABBSize() glm.Vec3f {
	var maxVec = m.vertices[0]
	var minVec = m.vertices[0]
	for _, v := range m.vertices {
		maxVec = maxVec.Max(v)
		minVec = minVec.Min(v)
	}

	return maxVec.Sub(minVec)
}

func (m *Mesh) HasAttribute(attr VertexAttribute) bool {
	ok := m.attributes.Has(attr)
	return ok
}

func (m *Mesh) ComputeNormals() {
	m.MergeVertices()

	// TODO better without merge, also differentiate to ComputeFlatNormals()

	normals := make([]glm.Vec3f, len(m.vertices))
	normalCounts := make([]uint32, len(m.vertices))

	for i := 0; i < len(m.indices); i += 3 {
		iA := m.indices[i]
		iB := m.indices[i+1]
		iC := m.indices[i+2]

		a := m.vertices[iA]
		b := m.vertices[iB]
		c := m.vertices[iC]

		normal := b.Sub(a).Cross(a.Sub(c)).Normalize()
		normals[iA] = normals[iA].Add(normal)
		normals[iB] = normals[iB].Add(normal)
		normals[iC] = normals[iC].Add(normal)

		normalCounts[iA] += 1
		normalCounts[iB] += 1
		normalCounts[iC] += 1
	}

	for idx, count := range normalCounts {
		normals[idx] = normals[idx].
			Scale(1.0 / max(1.0, float32(count))).
			Normalize()
	}

	m.WithAttributes(VertexAttributeNormal, wgpu.ToBytes(normals))
}

func (m *Mesh) MergeVertices() bool {
	if len(m.attributes) > 0 || len(m.morphTargets) > 0 {
		return false
	}

	// TODO handle vertex attributes or reject MergeVertices
	//  if we can not handle them correctly

	var byVertex = map[glm.Vec3f]uint32{}
	var newIndices []uint32
	var newVertices []glm.Vec3f

	for _, idx := range m.indices {
		vertex := m.vertices[idx]

		idx, seen := byVertex[vertex]
		if seen {
			newIndices = append(newIndices, idx)
			continue
		}

		// new vertex at new index
		idx = uint32(len(newVertices))
		newIndices = append(newIndices, idx)
		newVertices = append(newVertices, vertex)

		// put into lookup table
		byVertex[vertex] = idx
	}

	m.indices = newIndices
	m.vertices = newVertices
	m.version += 1

	return true
}

func (m *Mesh) SmoothShade() {
	type vertexInfo struct {
		AccNormal glm.Vec3f
		Count     uint32
		Indices   []uint32
	}

	normalsAttr := m.attributes.Get(VertexAttributeNormal)
	if normalsAttr == nil {
		panic("no normals set")
	}

	normals := unsafe.Slice(
		(*glm.Vec3f)(unsafe.Pointer(unsafe.SliceData(normalsAttr.Value))),
		uintptr(len(normalsAttr.Value))/unsafe.Sizeof(glm.Vec3f{}),
	)

	infos := map[glm.Vec3f]vertexInfo{}

	for idx, vertex := range m.vertices {
		info := infos[vertex]

		infos[vertex] = vertexInfo{
			AccNormal: info.AccNormal.Add(normals[idx]),
			Count:     info.Count + 1,
			Indices:   append(info.Indices, uint32(idx)),
		}
	}

	for _, info := range infos {
		normal := info.AccNormal.
			Scale(1 / float32(info.Count)).
			Normalize()

		for _, idx := range info.Indices {
			normals[idx] = normal
		}
	}

	m.version += 1
}

func (m *Mesh) ComputeTangents() {
	if !m.HasAttribute(VertexAttributeNormal) {
		slog.Warn("Cannot calculate tangents without normals")
		return
	}

	if !m.HasAttribute(VertexAttributeUV) {
		slog.Warn("Cannot calculate tangents without UV coordinates")
		return
	}

	// TODO need to unmerge vertices first

	tangents := make([]glm.Vec4f, len(m.vertices))

	mikktspace.GenerateTangents(meshGeometry{
		Mesh:     m,
		Tangents: tangents,
	})

	m.WithAttributes(VertexAttributeTangentSpace, wgpu.ToBytes(tangents))
}

func RegularPolygon(radius float32, sides uint) *Mesh {
	// a regular polygon is actually just a circle
	return Circle(radius, sides)
}

func Circle(radius float32, resolution uint) *Mesh {
	size := glm.Vec2f{radius, radius}.Scale(2.0)
	return Ellipse(size, resolution)
}

func Ellipse(size glm.Vec2f, resolution uint) *Mesh {
	halfSize := size.Scale(0.5)

	indices := make([]uint32, 0, (resolution-2)*3)
	vertices := make([]glm.Vec3f, 0, resolution)
	uvs := make([]glm.Vec2f, 0, resolution)

	startAngle := glm.Rad(math.Pi / 2)
	step := (2 * math.Pi) / glm.Rad(resolution)

	for i := range resolution {
		theta := startAngle + glm.Rad(i)*step
		cos := f32.Cos(float32(theta))
		sin := f32.Sin(float32(theta))

		x := cos * halfSize[0]
		y := sin * halfSize[1]

		vertices = append(vertices, glm.Vec3f{x, y})

		uvs = append(uvs, glm.Vec2f{
			0.5 * (cos + 1.0),
			1.0 - 0.5*(sin+1.0),
		})
	}

	for i := uint32(1); i < uint32(resolution)-1; i++ {
		indices = append(indices, 0, i, i+1)
	}

	return MeshOf(indices, vertices).WithAttributes(VertexAttributeUV, wgpu.ToBytes(uvs))
}

func Rectangle(size glm.Vec2f) *Mesh {
	hw, hh := size.Scale(0.5).XY()

	vertices := [4]glm.Vec3f{
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

	return MeshOf(indices, vertices[:]).WithAttributes(VertexAttributeUV, wgpu.ToBytes(uvs[:]))
}

func VertexAttributesOf(attr VertexAttribute, value []byte) VertexAttributes {
	return VertexAttributes{
		{
			Attribute: attr,
			Value:     value,
		},
	}
}

func ConvexPolygon(vertices []glm.Vec3f) *Mesh {
	if len(vertices) < 3 {
		panic(errors.New("polygon must have at least 3 vertices"))
	}

	// create triangles for the polygon
	indices := make([]uint32, 0, (len(vertices)-2)*3)
	for i := uint32(1); i < uint32(len(vertices)-1); i++ {
		indices = append(indices, 0, i, i+1)
	}

	return MeshOf(indices, vertices)
}

// Polygon creates a Mesh from a (possibly concave) polygon. The polygon might
// contain holes. A best effort at triangulation is performed.
func Polygon(polygon []glm.Vec2f, holes ...[]glm.Vec2f) *Mesh {
	// glm.Vec2f is binary compatible with earcut.Point[float32], so we can
	// just cast the slice data accordingly without needing to copy the actual data
	ecPolygons := castVecsToEarcutPoints(polygon)

	var ecHoles [][]earcut.Point[float32]
	for _, hole := range holes {
		ecHoles = append(ecHoles, castVecsToEarcutPoints(hole))
	}

	points, indices := earcut.Triangulate(ecPolygons, ecHoles)

	var vertices []glm.Vec3f
	for _, point := range points {
		vertices = append(vertices, glm.Vec3f{point.X, point.Y})
	}

	return MeshOf(indices, vertices)
}

func castVecsToEarcutPoints(vecs []glm.Vec2f) []earcut.Point[float32] {
	data := unsafe.Pointer(unsafe.SliceData(vecs))
	points := (*earcut.Point[float32])(data)
	return unsafe.Slice(points, len(vecs))
}

// UnionDisjoint merges disjoint meshes into one. The meshes must not
// overlap for rendering not to break.
func UnionDisjoint(meshes ...*Mesh) *Mesh {
	var vertexCount, indexCount int
	for _, mesh := range meshes {
		vertexCount += len(mesh.vertices)
		indexCount += len(mesh.indices)
	}

	vertices := make([]glm.Vec3f, 0, vertexCount)
	indices := make([]uint32, 0, indexCount)

	for _, mesh := range meshes {
		offset := uint32(len(mesh.vertices))
		vertices = append(vertices, mesh.vertices...)

		for _, idx := range mesh.indices {
			indices = append(indices, idx+offset)
		}
	}

	return MeshOf(indices, vertices)
}

/*
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
