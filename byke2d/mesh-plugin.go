package byke2d

import (
	"log/slog"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

func pluginMesh(app *byke.App) {
	app.InsertResource(byke.InitFromWorld(meshCacheFromWorld))

	app.InsertResource(MaterialBindGroups{})

	app.AddSystems(Render, byke.System(prepareMesh2dBuffers).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareMesh3dBuffers).InSet(RenderPhasePrepareResources))
}

type ExtractedMesh struct {
	Mesh *Mesh

	Transform    glm.Mat4f
	RenderLayers RenderLayers
	Material     Material

	Skin     ExtractedSkin
	EntityId byke.EntityId
}

type ExtractedSkin struct {
	EntityId    byke.EntityId
	InverseBind []glm.Mat4f
	Joints      []byke.EntityId
}

func (s *ExtractedSkin) IsSet() bool {
	return s.EntityId != byke.NoEntityId
}

func prepareMesh2dBuffers(
	meshes byke.Query[*Mesh2d],
	meshCache *meshCache,
) {
	meshCache.Reset()

	for item := range meshes.Items() {
		mesh := item.Mesh
		meshCache.Upload(mesh)
	}
}

func prepareMesh3dBuffers(
	meshes byke.Query[struct {
		Mesh *Mesh3d
		Name byke.Option[byke.Name]
	}],
	meshCache *meshCache,
) {
	meshCache.Reset()

	for item := range meshes.Items() {
		mesh := item.Mesh.Mesh
		uploaded := meshCache.Upload(mesh)

		if uploaded {
			// TODO use a lifecycle hook to print this maybe?
			name := item.Name.Or(byke.Named("unknown")).Name
			slog.Info(
				"Uploaded mesh",
				slog.String("name", name),
				slog.Any("bbox", mesh.AABBSize()),
			)
		}
	}
}
