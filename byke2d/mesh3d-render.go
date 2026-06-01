package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh3d(app *byke.App) {
	app.InsertResource(ExtractedMesh3d{})
	app.InsertResource(mesh3dInstances{})

	app.AddSystems(Render, byke.System(queueMesh3dSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(prepareMesh3dInstances).InSet(RenderPhasePrepareBindGroups))
	app.AddSystems(Render, byke.System(clearExtractedMesh3dSystem).InSet(RenderPhaseCleanup))

	app.AddPlugin(PluginMaterial3d[StandardMaterial])
}

type ExtractedMesh3d struct {
	Meshes []ExtractedMesh
}

func extractMesh3dSystem[M Material](
	meshes *ExtractedMesh3d,
	meshQuery byke.Query[struct {
		Mesh         query.Ref[Mesh3d]
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

func clearExtractedMesh3dSystem(
	meshes *ExtractedMesh3d,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type mesh3dRenderPhaseItem struct{}

func queueMesh3dSystem(
	meshes *ExtractedMesh3d,
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		RenderLayers RenderLayers
		RenderPhase  *BinnedRenderPhase[Opaque]
	}],
) {
	for view := range viewsQuery.Items() {
		for idx := range meshes.Meshes {
			sp := &meshes.Meshes[idx]
			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				continue
			}

			renderItem := RenderItem{
				Type:           &mesh3dRenderPhaseItem{},
				Draw:           drawMesh3dBatch,
				ExtractedIndex: uint32(idx),
			}

			key := MeshKey{
				Type:     &mesh3dRenderPhaseItem{},
				Material: sp.Material,
				Mesh:     sp.Mesh,
			}

			view.RenderPhase.Append(renderItem, key)
		}
	}
}

// mesh3dInstances stores the instance buffer for all per-instance
// data of the meshes
type mesh3dInstances struct {
	Buffer    *wgpu.Buffer
	Instances wgsl.InstanceWriter
}

func prepareMesh3dInstances(
	ctx *RenderContext,
	meshes *ExtractedMesh3d,
	pipelineCache *PipelineCache,
	meshInstances *mesh3dInstances,
	bindGroups *materialBindGroupCache,
	viewsQuery byke.Query[struct {
		_     byke.With[Camera]
		Phase *BinnedRenderPhase[Opaque]
	}],
) {
	instances := &meshInstances.Instances
	instances.Clear()

	for view := range viewsQuery.Items() {
		if view.Phase.IsEmpty() {
			continue
		}

		for key, batch := range view.Phase.Batches() {
			if len(batch) == 0 {
				continue
			}

			key, ok := key.(MeshKey)
			if !ok {
				continue
			}

			_, isMesh := key.Type.(*mesh3dRenderPhaseItem)

			if !isMesh {
				// this batch is not a mesh
				continue
			}

			// create a bindgroup for the material
			if _, ok := bindGroups.Get(key.Material); !ok {
				bindGroup := createMaterialBindGroup(ctx, pipelineCache, key.Material)
				bindGroups.Add(key.Material, bindGroup)
			}

			batch[0].BatchBegin = uint32(instances.InstanceCount())
			batch[0].BatchCount = uint32(len(batch))

			for _, item := range batch {
				mesh := &meshes.Meshes[item.ExtractedIndex]

				// write sprite vertex data
				writeMeshInstanceValues(instances, mesh)
			}
		}
	}

	// upload buffer to gpu
	instances.WriteTo(ctx, &meshInstances.Buffer)
}

func drawMesh3dBatch(world *byke.World, pass *wgpu.RenderPassEncoder, item RenderItem) (ok bool) {
	world.RunSystemWithInValue(drawMesh3dBatchSystem, RenderTask{
		Pass: pass,
		Item: item,
	})

	return true
}

func drawMesh3dBatchSystem(
	viewBindGroup ViewBindGroup,
	lightsBindGroup LightsBindGroup,
	pipelines *PipelineCache,
	task byke.In[RenderTask],
	meshes *ExtractedMesh3d,
	instances *mesh3dInstances,
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

	pipelineConfig := mesh3dPipelineConfig{
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
	pass.SetBindGroup(1, lightsBindGroup.BindGroup, nil)
	pass.SetBindGroup(2, bindGroup, nil)

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
