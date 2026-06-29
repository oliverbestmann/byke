package byke2d

import (
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/mikktspace-go"
)

type meshGeometry struct {
	Tangents []glm.Vec4f
	*Mesh
}

func (m meshGeometry) NumFaces() uint32 {
	return uint32(len(m.indices) / 3)
}

func (m meshGeometry) NumVerticesOfFace(_ mikktspace.Face) uint32 {
	return 3
}

func (m meshGeometry) Position(fv mikktspace.FaceVertex) mikktspace.Vec3 {
	return m.vertices[m.indices[vertexOffset(fv)]]
}

func (m meshGeometry) Normal(fv mikktspace.FaceVertex) mikktspace.Vec3 {
	attrValue := m.attributes.Get(VertexAttributeNormal).Value
	ptrToVec := (*glm.Vec3f)(unsafe.Pointer(&attrValue[0]))
	values := unsafe.Slice(ptrToVec, uintptr(len(attrValue))/unsafe.Sizeof(glm.Vec3f{}))
	return values[m.indices[vertexOffset(fv)]]
}

func (m meshGeometry) TexCoord(fv mikktspace.FaceVertex) mikktspace.Vec2 {
	attrValue := m.attributes.Get(VertexAttributeUV).Value
	ptrToVec := (*glm.Vec2f)(unsafe.Pointer(&attrValue[0]))
	values := unsafe.Slice(ptrToVec, uintptr(len(attrValue))/unsafe.Sizeof(glm.Vec2f{}))
	return values[m.indices[vertexOffset(fv)]]
}

func (m meshGeometry) SetTangent(fv mikktspace.FaceVertex, tangentSpace mikktspace.TangentSpace, ok bool) {
	m.Tangents[m.indices[vertexOffset(fv)]] = tangentSpace.EncodedTangent()
}

func vertexOffset(fv mikktspace.FaceVertex) uint32 {
	return uint32(fv.Face())*3 + uint32(fv.Vertex())
}
