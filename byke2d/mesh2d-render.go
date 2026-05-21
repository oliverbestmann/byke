package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh2d(app *byke.App) {
	app.InsertResource(ExtractedMeshes{})
	app.InsertResource(meshInstances{})

	app.InsertResource(byke.InitFromWorld[mesh2dCache]())
	app.InsertResource(byke.InitFromWorld[Pipelines[mesh2dPipelineConfig]]())

	app.AddSystems(PreRender, meshSetIdOnChangeSystem)
	app.AddSystems(Render, byke.System(extractMeshesSystem).InSet(RenderPhaseExtract))
	app.AddSystems(Render, byke.System(queueMeshesSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(prepareMesh2dBuffers).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareMeshInstances).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(clearExtractedMeshesSystem).InSet(RenderPhaseCleanup))
}

type ExtractedMesh struct {
	MeshId uint32

	Texture *Texture

	Indices  []uint32
	Vertices []glm.Vec2f

	// TODO other attributes
	Colors []Color

	// optional custom shader definition to replace or extend the
	// mesh default shader.
	CustomShader *ShaderDef

	Transform    glm.Mat4f
	Color        Color
	RenderLayers RenderLayers
}

type ExtractedMeshes struct {
	Meshes []ExtractedMesh
}

func meshSetIdOnChangeSystem(
	meshQuery byke.Query[struct {
		_    byke.Changed[Mesh2d]
		Mesh query.Ref[Mesh2d]
	}],

	meshId *byke.Local[uint32],
) {
	for mesh := range meshQuery.Items() {
		mesh := mesh.Mesh.Get()
		if mesh.id == 0 {
			meshId.Value += 1
			mesh.id = meshId.Value
		}
	}
}

func extractMeshesSystem(
	meshes *ExtractedMeshes,
	meshQuery byke.Query[struct {
		Mesh         query.Ref[Mesh2d]
		Transform    GlobalTransform
		MeshColor    byke.Option[MeshColor]
		RenderLayers byke.Option[RenderLayers]
		CustomShader byke.Option[CustomShader]
		Visibility   ComputedVisibility
	}],
) {
	for item := range meshQuery.Items() {
		if !item.Visibility.Visible {
			continue
		}

		mesh := item.Mesh.Value

		meshes.Meshes = append(meshes.Meshes, ExtractedMesh{
			MeshId:       mesh.id,
			Indices:      mesh.Indices,
			Vertices:     mesh.Vertices,
			Colors:       mesh.Colors,
			CustomShader: item.CustomShader.OrZero().Shader,
			Transform:    item.Transform.Affine,
			Color:        item.MeshColor.OrZero().Color,
			RenderLayers: item.RenderLayers.Or(renderLayerZero),
		})
	}
}

func clearExtractedMeshesSystem(
	meshes *ExtractedMeshes,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type meshRenderPhaseItem struct{}

func queueMeshesSystem(
	meshes *ExtractedMeshes,
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		RenderLayers RenderLayers
		RenderPhase  *RenderPhase
	}],
) {
	for view := range viewsQuery.Items() {
		for idx := range meshes.Meshes {
			sp := &meshes.Meshes[idx]
			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				continue
			}

			view.RenderPhase.Append(RenderPhaseItem{
				Type:           &meshRenderPhaseItem{},
				Draw:           drawMeshBatch,
				SortValue:      sp.Transform.TranslateZ(),
				ExtractedIndex: uint32(idx),
			})
		}
	}
}

func prepareMesh2dBuffers(
	meshes *ExtractedMeshes,
	meshCache *mesh2dCache,
) {
	meshCache.Reset()

	for idx := range meshes.Meshes {
		mesh := &meshes.Meshes[idx]
		meshCache.Upload(mesh.MeshId, mesh.Vertices, mesh.Indices, mesh.Colors)
	}
}

type meshInstances struct {
	Buffer    *wgpu.Buffer
	Instances wgsl.InstanceWriter
}

func prepareMeshInstances(
	ctx *RenderContext,
	viewsQuery byke.Query[struct {
		_     byke.With[Camera]
		Phase RenderPhase
	}],
	meshes *ExtractedMeshes,
	meshInstances *meshInstances,
) {
	instances := &meshInstances.Instances
	instances.Clear()

	for view := range viewsQuery.Items() {
		if view.Phase.IsEmpty() {
			continue
		}

		var current *RenderPhaseItem
		var currentMesh *ExtractedMesh

		for idx := range view.Phase.Len() {
			item := view.Phase.Get(idx)

			_, isMesh := item.Type.(*meshRenderPhaseItem)

			if !isMesh {
				// not a mesh, end the current batch,
				current = nil
				currentMesh = nil
				continue
			}

			itemMesh := &meshes.Meshes[item.ExtractedIndex]

			//goland:noinspection GoMaybeNil
			if current == nil ||
				currentMesh.MeshId != itemMesh.MeshId ||
				currentMesh.Texture != itemMesh.Texture ||
				currentMesh.CustomShader != itemMesh.CustomShader {

				// we begin a new mesh batch here
				current = item
				currentMesh = itemMesh

				// record begin of batch
				current.BatchBegin = uint32(instances.InstanceCount())
				current.BatchCount = 0
			}

			// write sprite vertex data
			writeMeshInstanceValues(instances, itemMesh)
			current.BatchCount += 1
		}
	}

	// upload buffer to gpu
	instances.WriteTo(ctx, &meshInstances.Buffer)
}

func writeMeshInstanceValues(instances *wgsl.InstanceWriter, mesh *ExtractedMesh) {
	instances.StartNew(64)
	instances.AppendVec3f(mesh.Transform.Column(0).Truncate())
	instances.AppendVec3f(mesh.Transform.Column(1).Truncate())
	instances.AppendVec3f(mesh.Transform.Column(2).Truncate())
	instances.AppendVec3f(mesh.Transform.Column(3).Truncate())
	instances.AppendVec4f(mesh.Color.ToVec())
}

func drawMeshBatch(world *byke.World, pass *wgpu.RenderPassEncoder, item RenderPhaseItem) (ok bool) {
	world.RunSystemWithInValue(drawMeshBatchSystem, RenderTask{
		Pass: pass,
		Item: item,
	})

	return true
}

func drawMeshBatchSystem(
	viewBindGroup ViewBindGroup,
	pipelines Pipelines[mesh2dPipelineConfig],
	task byke.In[RenderTask],
	meshes *ExtractedMeshes,
	instances *meshInstances,
	meshCache *mesh2dCache,
	viewQuery ViewQuery[struct {
		ViewTarget         *ViewTarget
		ViewUniformsOffset DynamicOffset[ViewUniforms]
	}],
) {
	view := viewQuery.Get()

	pass := task.Value.Pass
	item := task.Value.Item

	mesh := meshes.Meshes[item.ExtractedIndex]

	buf, ok := meshCache.Get(mesh.MeshId)
	if !ok {
		// mesh not in cache, broken?
		panic("mesh data not in cache")
	}

	indexCount := uint32(len(mesh.Indices))

	pipeline := pipelines.Specialize(mesh2dPipelineConfig{
		Format:      view.ViewTarget.Format,
		SampleCount: view.ViewTarget.SampleCount,
		HasColors:   buf.Colors != nil,
	})

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, viewBindGroup.BindGroup, []uint32{view.ViewUniformsOffset.Offset})
	pass.SetVertexBuffer(0, instances.Buffer, 0, wgpu.WholeSize)
	pass.SetVertexBuffer(1, buf.Vertex, 0, wgpu.WholeSize)

	if buf.Colors != nil {
		pass.SetVertexBuffer(2, buf.Colors, 0, wgpu.WholeSize)
	}

	pass.SetIndexBuffer(buf.Indices, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
	pass.DrawIndexed(indexCount, item.BatchCount, 0, 0, item.BatchBegin)
}
