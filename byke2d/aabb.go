package byke2d

import (
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

// OutsideClip reports whether the box is completely outside the view, given the matrix from
// the box's local space to clip space. Only the side planes and the camera plane count, not
// near and far depth: this works with reversed depth and keeps shadow casters in front of the light.
func (a AABB) OutsideClip(localToClip glm.Mat4f) bool {
	var left, right, bottom, top, behind int

	for idx := range 8 {
		corner := glm.Vec4f{a.Min[0], a.Min[1], a.Min[2], 1}
		if idx&1 != 0 {
			corner[0] = a.Max[0]
		}
		if idx&2 != 0 {
			corner[1] = a.Max[1]
		}
		if idx&4 != 0 {
			corner[2] = a.Max[2]
		}

		c := localToClip.Transform(corner)
		if c[0] < -c[3] {
			left++
		}
		if c[0] > c[3] {
			right++
		}
		if c[1] < -c[3] {
			bottom++
		}
		if c[1] > c[3] {
			top++
		}
		if c[3] <= 0 {
			behind++
		}
	}

	return left == 8 || right == 8 || bottom == 8 || top == 8 || behind == 8
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
