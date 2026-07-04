package byke2d

import (
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/mikktspace-go"
)

type meshMikktspaceAdapter struct {
	Indices  []uint32
	Vertices []glm.Vec3f
	Normals  []glm.Vec3f
	UVs      []glm.Vec2f

	Tangents []glm.Vec4f
}

func (m meshMikktspaceAdapter) NumFaces() uint32 {
	return uint32(len(m.Indices) / 3)
}

func (m meshMikktspaceAdapter) NumVerticesOfFace(_ mikktspace.Face) uint32 {
	return 3
}

func (m meshMikktspaceAdapter) Position(fv mikktspace.FaceVertex) mikktspace.Vec3 {
	return m.Vertices[m.Indices[vertexOffset(fv)]]
}

func (m meshMikktspaceAdapter) Normal(fv mikktspace.FaceVertex) mikktspace.Vec3 {
	return m.Normals[m.Indices[vertexOffset(fv)]]
}

func (m meshMikktspaceAdapter) TexCoord(fv mikktspace.FaceVertex) mikktspace.Vec2 {
	return m.UVs[m.Indices[vertexOffset(fv)]]
}

func (m meshMikktspaceAdapter) SetTangent(fv mikktspace.FaceVertex, tangentSpace mikktspace.TangentSpace, ok bool) {
	m.Tangents[m.Indices[vertexOffset(fv)]] = tangentSpace.EncodedTangent()
}

func vertexOffset(fv mikktspace.FaceVertex) uint32 {
	return uint32(fv.Face())*3 + uint32(fv.Vertex())
}
