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
