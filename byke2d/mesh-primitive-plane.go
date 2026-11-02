package byke2d

import (
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func Plane() *Mesh {
	return PlaneWithSize(glm.V2f(1, 1))
}

func PlaneWithSize(size glm.Vec2f) *Mesh {
	hx := size.X / 2
	hz := size.Y / 2

	vertices := []glm.Vec3f{
		glm.V3f(-hx, 0, -hz),
		glm.V3f(hx, 0, -hz),
		glm.V3f(hx, 0, hz),
		glm.V3f(-hx, 0, hz),
	}

	indices := []uint32{
		0, 3, 1,
		1, 3, 2,
	}

	uvs := []glm.Vec2f{
		glm.V2f(0.0, 0.0), glm.V2f(0.0, 1.0), glm.V2f(1.0, 0.0), glm.V2f(1.0, 1.0),
	}

	normals := []glm.Vec3f{
		glm.V3f(0.0, 1.0, 0.0),
		glm.V3f(0.0, 1.0, 0.0),
		glm.V3f(0.0, 1.0, 0.0),
		glm.V3f(0.0, 1.0, 0.0),
	}

	return MeshOf(indices, vertices).
		WithAttributes(VertexAttributeUV, uvs).
		WithAttributes(VertexAttributeNormal, normals)
}
