package byke2d

import (
	"math"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

var _ = byke.ValidateComponent[AABB]()

type AABB struct {
	byke.ComparableComponent[AABB]
	Min glm.Vec3f
	Max glm.Vec3f
}

func (a AABB) Center() glm.Vec3f {
	return a.Min.Add(a.Max).Scale(0.5)
}

// abs32 compiles to a single FABS/ANDPS, unlike max(v, -v) or a sign bit mask.
func abs32(v float32) float32 {
	return float32(math.Abs(float64(v)))
}

// Transformed returns the center and half size of the axis aligned box in the space of m that
// encloses this box. It is larger than the box itself if m rotates.
func (a *AABB) Transformed(m *glm.Mat4f) (center, half glm.Vec3f) {
	return m.TransformBox(a.Center(), a.Max.Sub(a.Min).Scale(0.5))
}

// Frustum holds the planes of a view: left, right, bottom, top and the camera plane.
// Points inside have a positive distance to every plane. Near and far depth do not count:
// this works with reversed depth and keeps shadow casters in front of the light.
type Frustum [5]glm.Vec4f

// FrustumFromClip extracts the planes from the world to clip matrix (Gribb and Hartmann).
func FrustumFromClip(m glm.Mat4f) Frustum {
	row := func(idx int) glm.Vec4f {
		return glm.V4f(m[0][idx], m[1][idx], m[2][idx], m[3][idx])
	}

	x, y, w := row(0), row(1), row(3)
	return Frustum{w.Add(x), w.Sub(x), w.Add(y), w.Sub(y), w}
}

// OutsideBox reports whether the box is completely outside of one of the planes.
func (f *Frustum) OutsideBox(center, half glm.Vec3f) bool {
	for _, p := range f {
		// the distance of the box corner furthest along the plane normal
		distance := p.X*center.X + p.Y*center.Y + p.Z*center.Z + p.W +
			abs32(p.X)*half.X + abs32(p.Y)*half.Y + abs32(p.Z)*half.Z

		if distance <= 0 {
			return true
		}
	}

	return false
}

func updateAABBOfChangedMeshSystem(
	commands *byke.Commands,
	meshesQuery byke.Query[struct {
		_        byke.Changed[MeshInstance]
		EntityId byke.EntityId
		Mesh     MeshInstance
	}],
) {
	for item := range meshesQuery.Items() {
		m := item.Mesh.Mesh
		minVec, maxVec := m.AABB()

		commands.Entity(item.EntityId).Insert(AABB{
			Min: minVec,
			Max: maxVec,
		})
	}
}
