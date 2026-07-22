package byke2d

import (
	"log/slog"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func pluginMesh(app *byke.App) {
	app.InitResource(meshAllocatorFromWorld)

	app.AddSystems(Render, byke.System(allocateMeshesSystem).InSet(RenderPhasePrepareResources))
}

// ExtractedMesh represents a mesh in the render pipeline after being extracted from ECS entities.
// It combines geometry, material, transform, and skin data needed for rendering.
type ExtractedMesh struct {
	// Mesh is the geometry data to be rendered.
	Mesh *Mesh

	// Transform is the object-to-world matrix positioning and orienting the mesh.
	Transform glm.Mat4f

	// Material defines the appearance of the mesh (colors, textures, shaders).
	Material Material

	// Skin contains skeletal animation data if this mesh is skinned; check IsSet() before using.
	Skin ExtractedSkin

	// EntityId is the entity this mesh was extracted from.
	EntityId byke.EntityId

	// RenderLayers specifies which render passes should include this mesh.
	RenderLayers RenderLayers
}

// ExtractedSkin contains skeletal animation data for a mesh.
type ExtractedSkin struct {
	// EntityId is the entity containing the skin definition.
	EntityId byke.EntityId

	// InverseBind are the inverse bind pose matrices that transform from model space to bone space.
	InverseBind []glm.Mat4f

	// Joints are the entity IDs of the skeleton joints that influence this mesh.
	Joints []byke.EntityId
}

// IsSet reports whether this skin has valid skeletal animation data.
func (s *ExtractedSkin) IsSet() bool {
	return s.EntityId != byke.NoEntityId
}

func allocateMeshesSystem(
	meshes byke.Query[struct {
		Mesh *MeshInstance
		Name byke.Option[byke.Name]
	}],
	meshAllocator *MeshAllocator,
) {
	for item := range meshes.Items() {
		mesh := item.Mesh.Mesh

		if meshAllocator.Alloc(mesh) {
			name := item.Name.Or(byke.Named("unknown")).Name
			slog.Debug(
				"Uploaded mesh",
				slog.String("name", name),
				slog.Any("bbox", mesh.AABBSize()),
			)
		}
	}
}
