package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh2d(app *byke.App) {
	app.InsertResource(ExtractedMesh2d{})
	app.InsertResource(mesh2dInstances{})

	app.AddSystems(Render, byke.System(queueMesh2dSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(prepareMesh2dInstances).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(clearExtractedMesh2dSystem).InSet(RenderPhaseCleanup))

	app.AddPlugin(PluginMaterial2d[ColorMaterial])
}

type ExtractedMesh struct {
	Mesh *Mesh

	Transform    glm.Mat4f
	RenderLayers RenderLayers
	Material     Material
}

type ExtractedMesh2d struct {
	Meshes []ExtractedMesh
}

func extractMesh2dSystem[M Material](
	meshes *ExtractedMesh2d,
	meshQuery byke.Query[struct {
		Mesh         query.Ref[Mesh2d]
		Transform    GlobalTransform
		Material     M
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
			Mesh:         mesh.Mesh,
			Transform:    item.Transform.Affine,
			Material:     item.Material,
			RenderLayers: item.RenderLayers.Or(renderLayerZero),
		})
	}
}

func clearExtractedMesh2dSystem(
	meshes *ExtractedMesh2d,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type mesh2dRenderPhaseItem struct{}

func queueMesh2dSystem(
	meshes *ExtractedMesh2d,
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		RenderLayers RenderLayers
		RenderPhase  *RenderPhase[Opaque]
	}],
) {
	for view := range viewsQuery.Items() {
		for idx := range meshes.Meshes {
			sp := &meshes.Meshes[idx]
			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				continue
			}

			view.RenderPhase.Append(RenderPhaseItem{
				Type:           &mesh2dRenderPhaseItem{},
				Draw:           drawMesh2dBatch,
				SortValue:      sp.Transform.TranslateZ(),
				ExtractedIndex: uint32(idx),
			})
		}
	}
}

// mesh2dInstances stores the instance buffer for all per-instance
// data of the meshes
type mesh2dInstances struct {
	Buffer    *wgpu.Buffer
	Instances wgsl.InstanceWriter
}

func prepareMesh2dInstances(
	ctx *RenderContext,
	meshes *ExtractedMesh2d,
	pipelineCache *PipelineCache,
	meshInstances *mesh2dInstances,
	bindGroups *materialBindGroupCache,
	viewsQuery byke.Query[struct {
		_     byke.With[Camera]
		Phase RenderPhase[Opaque]
	}],
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

			_, isMesh := item.Type.(*mesh2dRenderPhaseItem)

			if !isMesh {
				// not a mesh, end the current batch,
				current = nil
				currentMesh = nil
				continue
			}

			itemMesh := &meshes.Meshes[item.ExtractedIndex]

			//goland:noinspection GoMaybeNil
			if current == nil ||
				currentMesh.Mesh != itemMesh.Mesh ||
				currentMesh.Material != itemMesh.Material {

				// we begin a new mesh batch here
				current = item
				currentMesh = itemMesh

				// record begin of batch
				current.BatchBegin = uint32(instances.InstanceCount())
				current.BatchCount = 0

				// create a bindgroup for the material
				if _, ok := bindGroups.Get(itemMesh.Material); !ok {
					bindGroup := createMaterialBindGroup(ctx, pipelineCache, itemMesh.Material)
					bindGroups.Add(itemMesh.Material, bindGroup)
				}
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
	instances.StartNew(48)
	instances.AppendVec3f(mesh.Transform.Column(0).Truncate())
	instances.AppendVec3f(mesh.Transform.Column(1).Truncate())
	instances.AppendVec3f(mesh.Transform.Column(2).Truncate())
	instances.AppendVec3f(mesh.Transform.Column(3).Truncate())
}

func drawMesh2dBatch(world *byke.World, pass *wgpu.RenderPassEncoder, item RenderPhaseItem) (ok bool) {
	world.RunSystemWithInValue(drawMesh2dBatchSystem, RenderTask{
		Pass: pass,
		Item: item,
	})

	return true
}

func drawMesh2dBatchSystem(
	viewBindGroup ViewBindGroup,
	pipelines *PipelineCache,
	task byke.In[RenderTask],
	meshes *ExtractedMesh2d,
	instances *mesh2dInstances,
	meshCache *meshCache,
	bindGroupCache *materialBindGroupCache,
	viewQuery ViewQuery[struct {
		ViewTarget         *ViewTarget
		ViewUniformsOffset DynamicOffset[ViewUniforms]
	}],
) {
	view := viewQuery.Get()

	pass := task.Value.Pass
	item := task.Value.Item

	mesh := meshes.Meshes[item.ExtractedIndex]

	buf, ok := meshCache.Get(mesh.Mesh)
	if !ok {
		// mesh not in cache, broken?
		panic("mesh data not in cache")
	}

	indexCount := uint32(len(mesh.Mesh.indices))

	var layout []wgpu.BindGroupLayoutEntry
	layout = append(layout, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false))
	layout = append(layout, mesh.Material.BindingsLayout()...)

	pipelineConfig := mesh2dPipelineConfig{
		Format:           view.ViewTarget.Format,
		SampleCount:      view.ViewTarget.SampleCount,
		Shader:           mesh.Material.Shader(),
		MaterialBindings: layout,
	}

	for idx := range buf.Attributes {
		// tell the pipeline about the attributes we want to use
		pipelineConfig.Attributes = append(
			pipelineConfig.Attributes,
			buf.Attributes[idx].Attribute,
		)
	}

	pipeline := pipelines.Specialize(pipelineConfig)

	bindGroup, _ := bindGroupCache.Get(mesh.Material)

	pass.SetPipeline(pipeline.Get())

	pass.SetBindGroup(0, viewBindGroup.BindGroup, []uint32{view.ViewUniformsOffset.Offset})
	pass.SetBindGroup(1, bindGroup, nil)

	pass.SetVertexBuffer(0, instances.Buffer, 0, wgpu.WholeSize)
	pass.SetVertexBuffer(1, buf.Vertex, 0, wgpu.WholeSize)

	// set vertex buffers for other attributes
	for idx := range buf.Attributes {
		buffer := buf.Attributes[idx].Buffer
		pass.SetVertexBuffer(uint32(2+idx), buffer, 0, wgpu.WholeSize)
	}

	pass.SetIndexBuffer(buf.Indices, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
	pass.DrawIndexed(indexCount, item.BatchCount, 0, 0, item.BatchBegin)
}
