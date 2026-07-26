package byke2d

import (
	"math"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"golang.org/x/mobile/exp/f32"
)

// Sphere creates a mesh representing a UV sphere with the given radius. The number of
// sectors (longitude subdivisions) and stacks (latitude subdivisions) control the resolution.
//
// Largely inspired from http://www.songho.ca/opengl/gl_sphere.html
func Sphere(radius float32, sectors, stacks int) *Mesh {
	fSectors := float32(sectors)
	fStacks := float32(stacks)
	lengthInv := 1.0 / radius
	sectorStep := 2.0 * float32(math.Pi) / fSectors
	stackStep := float32(math.Pi) / fStacks

	nVertices := (stacks + 1) * (sectors + 1)
	vertices := make([]glm.Vec3f, 0, nVertices)
	normals := make([]glm.Vec3f, 0, nVertices)
	uvs := make([]glm.Vec2f, 0, nVertices)
	indices := make([]uint32, 0, nVertices*2*3)

	for i := 0; i <= stacks; i++ {
		stackAngle := float32(math.Pi)/2 - float32(i)*stackStep
		xy := radius * f32.Cos(stackAngle)
		z := radius * f32.Sin(stackAngle)

		for j := 0; j <= sectors; j++ {
			sectorAngle := float32(j) * sectorStep
			x := xy * f32.Cos(sectorAngle)
			y := xy * f32.Sin(sectorAngle)

			vertices = append(vertices, glm.Vec3f{x, y, z})
			normals = append(normals, glm.Vec3f{x * lengthInv, y * lengthInv, z * lengthInv})
			uvs = append(uvs, glm.Vec2f{float32(j) / fSectors, float32(i) / fStacks})
		}
	}

	// indices
	//  k1--k1+1
	//  |  / |
	//  | /  |
	//  k2--k2+1
	for i := 0; i < stacks; i++ {
		k1 := uint32(i * (sectors + 1))
		k2 := k1 + uint32(sectors+1)

		for j := 0; j < sectors; j++ {
			if i != 0 {
				indices = append(indices, k1, k2, k1+1)
			}
			if i != stacks-1 {
				indices = append(indices, k1+1, k2, k2+1)
			}

			k1 += 1
			k2 += 1
		}
	}

	return MeshOf(indices, vertices).
		WithAttributes(VertexAttributeNormal, normals).
		WithAttributes(VertexAttributeUV, uvs)
}
