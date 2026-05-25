package byke2d

import "github.com/oliverbestmann/byke"

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
	meshes byke.Query[*Mesh3d],
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
