package byke2d

import (
	"log/slog"

	"github.com/oliverbestmann/byke"
)

func pluginMesh(app *byke.App) {
	app.InsertResource(byke.InitFromWorld(meshCacheFromWorld))

	app.InsertResource(materialBindGroupCache{})

	app.AddSystems(Render, byke.System(prepareMesh2dBuffers).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareMesh3dBuffers).InSet(RenderPhasePrepareResources))
}

func prepareMesh2dBuffers(
	meshes byke.Query[*Mesh2d],
	meshCache *meshCache,
) {
	meshCache.Reset()

	for item := range meshes.Items() {
		mesh := item.Mesh
		forceUpload := mesh.requireUpload()
		meshCache.Upload(mesh, forceUpload)
		mesh.markUploaded()
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
		forceUpload := mesh.requireUpload()
		uploaded := meshCache.Upload(mesh, forceUpload)
		mesh.markUploaded()

		if uploaded {
			// TODO use a lifecycle hook to print this maybe?
			name := item.Name.Or(byke.Named("unknown")).Name
			slog.Debug(
				"Mesh bounding box",
				slog.String("name", name),
				slog.Any("bbox", mesh.AABBSize()),
			)
		}
	}
}
