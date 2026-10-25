package byke2d

import (
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func Cube() *Mesh {
	return CubeWithSize(glm.Vec3f{1, 1, 1})
}

func CubeWithSize(size glm.Vec3f) *Mesh {
	hx := size[0] / 2
	hy := size[1] / 2
	hz := size[2] / 2

	vertices := []glm.Vec3f{
		// top (facing towards +y)
		{-hx, hy, -hz},
		{hx, hy, -hz},
		{hx, hy, hz},
		{-hx, hy, hz},
		// bottom   (-y)
		{-hx, -hy, -hz},
		{hx, -hy, -hz},
		{hx, -hy, hz},
		{-hx, -hy, hz},
		// right    (+x)
		{hx, -hy, -hz},
		{hx, -hy, hz},
		{hx, hy, hz},
		{hx, hy, -hz},
		// left     (-x)
		{-hx, -hy, -hz},
		{-hx, -hy, hz},
		{-hx, hy, hz},
		{-hx, hy, -hz},
		// back     (+z)
		{-hx, -hy, hz},
		{-hx, hy, hz},
		{hx, hy, hz},
		{hx, -hy, hz},
		// forward  (-z)
		{-hx, -hy, -hz},
		{-hx, hy, -hz},
		{hx, hy, -hz},
		{hx, -hy, -hz},
	}

	indices := []uint32{
		0, 3, 1, 1, 3, 2, // triangles making up the top (+y) facing side.
		4, 5, 7, 5, 6, 7, // bottom (-y)
		8, 11, 9, 9, 11, 10, // right (+x)
		12, 13, 15, 13, 14, 15, // left (-x)
		16, 19, 17, 17, 19, 18, // back (+z)
		20, 21, 23, 21, 22, 23, // forward (-z)
	}

	// each face maps the full texture, upright when looking at the face from outside
	uvs := []glm.Vec2f{
		// top (+y)
		{0, 0}, {1, 0}, {1, 1}, {0, 1},
		// bottom (-y)
		{0, 1}, {1, 1}, {1, 0}, {0, 0},
		// right (+x)
		{1, 1}, {0, 1}, {0, 0}, {1, 0},
		// left (-x)
		{0, 1}, {1, 1}, {1, 0}, {0, 0},
		// back (+z)
		{0, 1}, {0, 0}, {1, 0}, {1, 1},
		// forward (-z)
		{1, 1}, {1, 0}, {0, 0}, {0, 1},
	}

	normals := []glm.Vec3f{
		// Normals for the top side (towards +y)
		{0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 1.0, 0.0},
		// Normals for the bottom side (towards -y)
		{0.0, -1.0, 0.0},
		{0.0, -1.0, 0.0},
		{0.0, -1.0, 0.0},
		{0.0, -1.0, 0.0},
		// Normals for the right side (towards +x)
		{1.0, 0.0, 0.0},
		{1.0, 0.0, 0.0},
		{1.0, 0.0, 0.0},
		{1.0, 0.0, 0.0},
		// Normals for the left side (towards -x)
		{-1.0, 0.0, 0.0},
		{-1.0, 0.0, 0.0},
		{-1.0, 0.0, 0.0},
		{-1.0, 0.0, 0.0},
		// Normals for the back side (towards +z)
		{0.0, 0.0, 1.0},
		{0.0, 0.0, 1.0},
		{0.0, 0.0, 1.0},
		{0.0, 0.0, 1.0},
		// Normals for the forward side (towards -z)
		{0.0, 0.0, -1.0},
		{0.0, 0.0, -1.0},
		{0.0, 0.0, -1.0},
		{0.0, 0.0, -1.0},
	}

	return MeshOf(indices, vertices).
		WithAttributes(VertexAttributeUV, uvs).
		WithAttributes(VertexAttributeNormal, normals)
}
