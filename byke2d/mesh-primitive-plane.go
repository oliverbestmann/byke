package byke2d

import (
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func Plane() *Mesh {
	return PlaneWithSize(glm.Vec2f{1, 1})
}

func PlaneWithSize(size glm.Vec2f) *Mesh {
	hx := size[0] / 2
	hz := size[1] / 2

	vertices := []glm.Vec3f{
		{-hx, 0, -hz},
		{hx, 0, -hz},
		{hx, 0, hz},
		{-hx, 0, hz},
	}

	indices := []uint32{
		0, 3, 1,
		1, 3, 2,
	}

	uvs := []glm.Vec2f{
		{0.0, 0.0}, {0.0, 1.0}, {1.0, 0.0}, {1.0, 1.0},
	}

	normals := []glm.Vec3f{
		{0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0},
	}

	return MeshOf(indices, vertices).
		WithAttributes(VertexAttributeUV, uvs).
		WithAttributes(VertexAttributeNormal, normals)
}
