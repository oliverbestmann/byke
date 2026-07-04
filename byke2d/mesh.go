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

	// Additional vertex attributes
	attributes VertexAttributes

	// The vertex layout
	layout VertexLayout

	// a mesh can contain multiple morph targets
	morphTargets [][]MorphAttributes

	// set to true if the mesh is uploaded to the gpu
	version uint32
}

func MeshOf(indices []uint32, vertices []glm.Vec3f) *Mesh {
	data := ValuesAsByteSlice(vertices)

	mesh := &Mesh{indices: indices, version: 1}
	mesh.attributes.Insert(VertexAttributePosition, data)
	mesh.updateVertexLayout()
	return mesh
}

func (m *Mesh) WithVertices(vertices []glm.Vec3f) *Mesh {
	data := ValuesAsByteSlice(vertices)
	return m.WithAttributes(VertexAttributePosition, data)
}

func (m *Mesh) WithAttributes(attr VertexAttribute, values []byte) *Mesh {
	valueCount := len(values) / int(attr.Size())
	if valueCount != m.VertexCount() {
		panic(fmt.Errorf("expected values for %d vertices", m.VertexCount()))
	}

	m.version += 1
	m.attributes.Insert(attr, values)
	m.updateVertexLayout()
	return m
}

func (m *Mesh) HasAttribute(attr VertexAttribute) bool {
	ok := m.attributes.Has(attr)
	return ok
}

// WithMorphTarget adds another morph target to this vertex
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

// Vertices returns the vertices of this Mesh. You should not modify the
// returned slice.
func (m *Mesh) Vertices() []glm.Vec3f {
	data := m.attributes.Get(VertexAttributePosition)
	return ByteSliceAsValues[glm.Vec3f](data.Value)
}

func (m *Mesh) ComputeUV(compute func(point glm.Vec3f) glm.Vec2f) {
	var uvs []glm.Vec2f

	for _, vertex := range m.Vertices() {
		uv := compute(vertex)
		uvs = append(uvs, uv)
	}

	m.WithAttributes(VertexAttributeUV, wgpu.ToBytes(uvs))
}

func (m *Mesh) VertexCount() int {
	return len(m.Vertices())
}

func (m *Mesh) MorphTargetCount() int {
	return len(m.morphTargets)
}

// Transform applies the given matrix to all vertices within this mesh
func (m *Mesh) Transform(tr glm.Mat4f) {
	vertices := m.Vertices()

	for idx := range vertices {
		vertices[idx] = tr.Transform3(vertices[idx])
	}
}

func (m *Mesh) AABBSize() glm.Vec3f {
	vertices := m.Vertices()

	var maxVec = vertices[0]
	var minVec = vertices[0]
	for _, v := range vertices {
		maxVec = maxVec.Max(v)
		minVec = minVec.Min(v)
	}

	return maxVec.Sub(minVec)
}

func (m *Mesh) ComputeNormals() {
	m.MergeVertices()

	// TODO better without merge, also differentiate to ComputeFlatNormals()

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

	m.WithAttributes(VertexAttributeNormal, wgpu.ToBytes(normals))
}

func (m *Mesh) MergeVertices() bool {
	if len(m.attributes) > 0 || len(m.morphTargets) > 0 {
		return false
	}

	// TODO handle vertex attributes or reject MergeVertices
	//  if we can not handle them correctly

	vertices := m.Vertices()

	var byVertex = map[glm.Vec3f]uint32{}
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

	data := ValuesAsByteSlice(newVertices)
	m.attributes.Insert(VertexAttributePosition, data)
	m.updateVertexLayout()

	m.indices = newIndices
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

	// TODO need to unmerge vertices first

	tangents := make([]glm.Vec4f, len(m.Vertices()))

	mikktspace.GenerateTangents(meshGeometry{
		Indices:  m.indices,
		Vertices: m.Vertices(),
		Normals:  ByteSliceAsValues[glm.Vec3f](normalsAttrs.Value),
		UVs:      ByteSliceAsValues[glm.Vec2f](uvsAttrs.Value),
		Tangents: tangents,
	})

	m.WithAttributes(VertexAttributeTangentSpace, wgpu.ToBytes(tangents))
}

func (m *Mesh) updateVertexLayout() {
	var attrs []VertexAttribute
	for _, attr := range m.attributes.Values() {
		attrs = append(attrs, attr.Attribute)
	}

	m.layout = NewVertexLayout(attrs)
}

func (m *Mesh) VertexLayout() VertexLayout {
	return m.layout
}

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

func VertexAttributesOf[T any](attr VertexAttribute, values []T) VertexAttributes {
	return VertexAttributes{
		{
			Attribute: attr,
			Value:     wgpu.ToBytes(values),
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
	return ValuesToValues[glm.Vec2f, earcut.Point[float32]](vecs)
}
