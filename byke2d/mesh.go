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

// Mesh represents a collection of vertices, indices, and vertex attributes that define
// 3D geometry. It supports flexible vertex attributes, morph targets for skeletal animation,
// and provides methods for geometric computations like normal and tangent generation.
type Mesh struct {
	// indices indexes into vertices to draw triangles from.
	// Length of indices must be a multiple of three.
	indices []uint32

	// Additional vertex attributes
	attributes VertexAttributes

	// The vertex layout
	layout VertexLayout

	// a mesh can contain multiple morph targets
	morphTargets [][]MorphAttributes

	// set to true if the mesh is uploaded to the gpu
	version uint32
}

// MeshOf creates a new mesh with the given indices and vertex positions.
func MeshOf(indices []uint32, vertices []glm.Vec3f) *Mesh {
	mesh := &Mesh{indices: indices, version: 1}
	mesh.attributes.Insert(VertexAttributePosition, vertices)
	mesh.updateVertexLayout()
	return mesh
}

// WithVertices updates the vertex positions of this mesh and returns the mesh for chaining.
func (m *Mesh) WithVertices(vertices []glm.Vec3f) *Mesh {
	return m.WithAttributes(VertexAttributePosition, vertices)
}

// WithAttributes adds or updates a vertex attribute for all vertices in this mesh.
// The values byte slice must contain exactly one attribute value per vertex.
// Returns the mesh for method chaining.
func (m *Mesh) WithAttributes[T any](attr VertexAttribute, values []T) *Mesh {
	if unsafeSizeOf[T]() != int(attr.Size()) {
		panic(fmt.Errorf("expected value size of %d, got %d", attr.Size(), unsafeSizeOf[T]()))
	}

	valueCount := len(values)
	if valueCount != m.VertexCount() {
		panic(fmt.Errorf("expected values for %d vertices, got %d", m.VertexCount(), valueCount))
	}

	m.version += 1
	m.attributes.Insert(attr, values)
	m.updateVertexLayout()
	return m
}

// HasAttribute reports whether this mesh has the specified vertex attribute.
func (m *Mesh) HasAttribute(attr VertexAttribute) bool {
	ok := m.attributes.Has(attr)
	return ok
}

// WithMorphTarget adds a morph target to the mesh. Each morph target provides
// alternative attributes for vertices to enable blend-shape animation.
// The target must have one attribute value per vertex in the mesh.
func (m *Mesh) WithMorphTarget(target []MorphAttributes) *Mesh {
	if len(target) != m.VertexCount() {
		panic(fmt.Errorf(
			"got %d morph attributes for %d vertices",
			len(target), m.VertexCount(),
		))
	}

	m.version += 1
	m.morphTargets = append(m.morphTargets, target)

	return m
}

// Vertices returns the vertex positions of this mesh. The returned slice is a view
// into the mesh's internal data and must not be modified; use WithVertices instead.
func (m *Mesh) Vertices() []glm.Vec3f {
	data := m.attributes.Get(VertexAttributePosition)
	return ByteSliceAsValues[glm.Vec3f](data.Value)
}

// ComputeUV calculates UV coordinates for all vertices using the provided function.
// The function receives a vertex position and should return its corresponding UV coordinate.
func (m *Mesh) ComputeUV(compute func(point glm.Vec3f) glm.Vec2f) {
	var uvs []glm.Vec2f

	for _, vertex := range m.Vertices() {
		uv := compute(vertex)
		uvs = append(uvs, uv)
	}

	m.WithAttributes(VertexAttributeUV, uvs)
}

// VertexCount returns the number of vertices in this mesh.
func (m *Mesh) VertexCount() int {
	return len(m.Vertices())
}

// MorphTargetCount returns the number of morph targets attached to this mesh.
func (m *Mesh) MorphTargetCount() int {
	return len(m.morphTargets)
}

// Transform applies the given transformation matrix to all vertices in this mesh.
func (m *Mesh) Transform(tr glm.Mat4f) {
	vertices := m.Vertices()

	for idx := range vertices {
		vertices[idx] = tr.Transform3(vertices[idx])
	}
}

// AABB returns the minimum and the maximum coordinate of
// the axis-aligned bounding box that contains all vertices.
func (m *Mesh) AABB() (min, max glm.Vec3f) {
	vertices := m.Vertices()

	maxVec := vertices[0]
	minVec := vertices[0]
	for _, v := range vertices {
		maxVec = maxVec.Max(v)
		minVec = minVec.Min(v)
	}

	return minVec, maxVec
}

// AABBSize returns the dimensions of the axis-aligned bounding box that contains all vertices.
func (m *Mesh) AABBSize() glm.Vec3f {
	minVec, maxVec := m.AABB()
	return maxVec.Sub(minVec)
}

// ComputeNormals calculates smooth vertex normals from the mesh geometry.
// Normals are computed per-vertex by averaging the face normals of all adjacent faces.
func (m *Mesh) ComputeNormals() {

	// TODO better without merge, also differentiate to ComputeFlatNormals()
	//  m.MergeVertices()

	vertices := m.Vertices()

	normals := make([]glm.Vec3f, len(vertices))
	normalCounts := make([]uint32, len(vertices))

	for i := 0; i < len(m.indices); i += 3 {
		iA := m.indices[i]
		iB := m.indices[i+1]
		iC := m.indices[i+2]

		a := vertices[iA]
		b := vertices[iB]
		c := vertices[iC]

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

	m.WithAttributes(VertexAttributeNormal, normals)
}

// MergeVertices removes duplicate vertices from the mesh, reducing memory usage and improving
// rendering performance. It returns true if vertices were merged, false if the mesh could not
// be merged (e.g., due to vertex attributes or morph targets being present).
func (m *Mesh) MergeVertices() bool {
	if len(m.attributes) > 0 || len(m.morphTargets) > 0 {
		return false
	}

	// TODO handle vertex attributes or reject MergeVertices
	//  if we can not handle them correctly

	vertices := m.Vertices()

	byVertex := map[glm.Vec3f]uint32{}
	var newIndices []uint32
	var newVertices []glm.Vec3f

	for _, idx := range m.indices {
		vertex := vertices[idx]

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

	m.attributes.Insert(VertexAttributePosition, newVertices)
	m.updateVertexLayout()

	m.indices = newIndices
	m.version += 1

	return true
}

// SmoothShade averages normals for vertices at the same position, creating smooth shading
// across faces that share vertices. This requires normals to already be computed.
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

	for idx, vertex := range m.Vertices() {
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

// ComputeTangents calculates tangent and bitangent vectors for all vertices.
// This requires both normals and UV coordinates to be present.
// It uses the mikktspace algorithm.
func (m *Mesh) ComputeTangents() {
	normalsAttrs := m.attributes.Get(VertexAttributeNormal)
	uvsAttrs := m.attributes.Get(VertexAttributeUV)

	if normalsAttrs == nil {
		slog.Warn("Cannot calculate tangents without normals")
		return
	}

	if uvsAttrs == nil {
		slog.Warn("Cannot calculate tangents without UV coordinates")
		return
	}

	tangents := make([]glm.Vec4f, len(m.Vertices()))

	mikktspace.GenerateTangents(meshMikktspaceAdapter{
		Indices:  m.indices,
		Vertices: m.Vertices(),
		Normals:  normalsAttrs.ValuesAs[glm.Vec3f](),
		UVs:      uvsAttrs.ValuesAs[glm.Vec2f](),
		Tangents: tangents,
	})

	m.WithAttributes(VertexAttributeTangentSpace, tangents)
}

func (m *Mesh) updateVertexLayout() {
	var attrs []VertexAttribute
	for _, attr := range m.attributes.Values() {
		attrs = append(attrs, attr.Attribute)
	}

	m.layout = NewVertexLayout(attrs)
}

// VertexLayout returns the layout descriptor for vertices in this mesh, which specifies
// what attributes are present and their formats and offsets.
func (m *Mesh) VertexLayout() VertexLayout {
	return m.layout
}

// WriteVerticesTo serializes all vertices to the given buffer in GPU-friendly interleaved format
// (position, normal, uv, etc.) and returns the extended buffer and vertex layout.
func (m *Mesh) WriteVerticesTo(buf []byte) ([]byte, VertexLayout) {
	layout := m.VertexLayout()

	prevSize := len(buf)
	expectedSize := m.VertexCount() * int(layout.Size())

	for idx := range uint32(m.VertexCount()) {
		for _, attr := range layout.Attributes {
			attrValue := m.attributes.Get(attr)
			if attrValue == nil {
				// should never happen
				panic(fmt.Errorf("attribute value is missing: %q", attr.Name))
			}

			buf = appendVertexRawTo(buf, attrValue.Value, attr.Format, idx)
		}
	}

	written := len(buf) - prevSize
	if written != expectedSize {
		panic(fmt.Errorf("expected to write %d bytes, got %d", expectedSize, written))
	}

	return buf, layout
}

func appendVertexRawTo(target []byte, values []byte, format wgpu.VertexFormat, idx uint32) []byte {
	slice := values[idx*format.ByteSize() : (idx+1)*format.ByteSize()]
	return append(target, slice...)
}

// RegularPolygon creates a mesh representing a regular polygon (equilateral triangle, square, etc.).
func RegularPolygon(radius float32, sides uint) *Mesh {
	// a regular polygon is actually just a circle
	return Circle(radius, sides)
}

// Circle creates a mesh representing a circle with the given radius and vertex resolution.
func Circle(radius float32, resolution uint) *Mesh {
	size := glm.Vec2f{radius, radius}.Scale(2.0)
	return Ellipse(size, resolution)
}

// Ellipse creates a mesh representing an ellipse with the given size and vertex resolution.
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

	return MeshOf(indices, vertices).WithAttributes(VertexAttributeUV, uvs)
}

// Rectangle creates a mesh representing a rectangle with the given size, centered at the origin.
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

	return MeshOf(indices, vertices[:]).WithAttributes(VertexAttributeUV, uvs[:])
}

// ConvexPolygon creates a mesh from a convex polygon by triangulating from its first vertex.
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

// Polygon creates a mesh from a 2D polygon using robust triangulation. The polygon can be
// concave and may contain holes. Each hole is a sequence of 2D points defining its boundary.
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
	return ValuesAsValues[glm.Vec2f, earcut.Point[float32]](vecs)
}

func unsafeSizeOf[T any]() int {
	var tZero T
	return int(unsafe.Sizeof(tZero))
}
