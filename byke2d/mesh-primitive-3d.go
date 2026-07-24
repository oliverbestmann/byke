package byke2d

import (
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func Cube() *Mesh {
	vertices := []glm.Vec3f{
		// top (facing towards +y)
		{-0.5, 0.5, -0.5}, // vertex with index0
		{0.5, 0.5, -0.5},  // vertex with index1
		{0.5, 0.5, 0.5},   // etc. until 3
		{-0.5, 0.5, 0.5},
		// bottom   (-y)
		{-0.5, -0.5, -0.5},
		{0.5, -0.5, -0.5},
		{0.5, -0.5, 0.5},
		{-0.5, -0.5, 0.5},
		// right    (+x)
		{0.5, -0.5, -0.5},
		{0.5, -0.5, 0.5},
		{0.5, 0.5, 0.5}, // This vertex is at the same position as vertex with index 2, but they'll have different UV and norml
		{0.5, 0.5, -0.5},
		// left     (-x)
		{-0.5, -0.5, -0.5},
		{-0.5, -0.5, 0.5},
		{-0.5, 0.5, 0.5},
		{-0.5, 0.5, -0.5},
		// back     (+z)
		{-0.5, -0.5, 0.5},
		{-0.5, 0.5, 0.5},
		{0.5, 0.5, 0.5},
		{0.5, -0.5, 0.5},
		// forward  (-z)
		{-0.5, -0.5, -0.5},
		{-0.5, 0.5, -0.5},
		{0.5, 0.5, -0.5},
		{0.5, -0.5, -0.5},
	}

	indices := []uint32{
		0, 3, 1, 1, 3, 2, // triangles making up the top (+y) facing side.
		4, 5, 7, 5, 6, 7, // bottom (-y)
		8, 11, 9, 9, 11, 10, // right (+x)
		12, 13, 15, 13, 14, 15, // left (-x)
		16, 19, 17, 17, 19, 18, // back (+z)
		20, 21, 23, 21, 22, 23, // forward (-z)
	}

	uvs := []glm.Vec2f{
		// Assigning the UV coords for the top side.
		{0.0, 0.2}, {0.0, 0.0}, {1.0, 0.0}, {1.0, 0.2},
		// Assigning the UV coords for the bottom side.
		{0.0, 0.45}, {0.0, 0.25}, {1.0, 0.25}, {1.0, 0.45},
		// Assigning the UV coords for the right side.
		{1.0, 0.45}, {0.0, 0.45}, {0.0, 0.2}, {1.0, 0.2},
		// Assigning the UV coords for the left side.
		{1.0, 0.45}, {0.0, 0.45}, {0.0, 0.2}, {1.0, 0.2},
		// Assigning the UV coords for the back side.
		{0.0, 0.45}, {0.0, 0.2}, {1.0, 0.2}, {1.0, 0.45},
		// Assigning the UV coords for the forward side.
		{0.0, 0.45}, {0.0, 0.2}, {1.0, 0.2}, {1.0, 0.45},
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
