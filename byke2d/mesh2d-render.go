package byke2d

import (
	"cmp"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh2d(app *byke.App) {
	app.InsertResource(ExtractedMeshes2d{})
	app.InsertResource(mesh2dInstances{})

	app.AddSystems(Render, byke.System(queueMesh2dSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(prepareMesh2dInstances).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(clearExtractedMesh2dSystem).InSet(RenderPhaseCleanup))

	app.AddPlugin(PluginMaterial2d[ColorMaterial])
}

type ExtractedMeshes2d struct {
	Meshes []ExtractedMesh
}

func extractMesh2dSystem[M Material](
	meshes *ExtractedMeshes2d,
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
	meshes *ExtractedMeshes2d,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type mesh2dRenderPhaseItem struct{}

func queueMesh2dSystem(
	meshes *ExtractedMeshes2d,
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
				Type:           &mesh2dRenderPhaseItem{},
				Draw:           drawMesh2dBatch,
				ExtractedIndex: uint32(idx),
			}

			key := &MeshKey{
				Type:      &mesh2dRenderPhaseItem{},
				MatKey:    sp.Material.BindGroupKey(),
				MatType:   reflect.TypeOf(sp.Material),
				LayoutKey: sp.Mesh.VertexLayout().Key(),
			}

			view.RenderPhase.Append(renderItem, key)
		}
	}
}

// mesh2dInstances stores the instance buffer for all per-instance
// data of the meshes
type mesh2dInstances struct {
	Buffer    *wgpu.Buffer
	Instances wgsl.InstanceWriter
}

type MeshKey struct {
	Type      any
	MatType   reflect.Type
	MatKey    MaterialBindGroupKey
	LayoutKey VertexLayoutKey
}

func (m *MeshKey) CompareTo(other any) int {
	o, ok := other.(*MeshKey)
	if !ok {
		return compareByType(m, other)
	}

	return cmp.Or(
		compareByType(m.Type, o.Type),
		compareType(m.MatType, o.MatType),
		cmp.Compare(m.LayoutKey, o.LayoutKey),
		cmp.Compare(m.MatKey.SortValue(), o.MatKey.SortValue()),
	)
}

func prepareMesh2dInstances(
	ctx *RenderContext,
	meshes *ExtractedMeshes2d,
	meshInstances *mesh2dInstances,
	materialUniforms *MaterialUniforms[ColorMaterial],
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

			key, ok := key.(*MeshKey)
			if !ok {
				continue
			}

			_, isMesh := key.Type.(*mesh2dRenderPhaseItem)
			if !isMesh {
				// this batch is not a mesh
				continue
			}

			batch[0].BatchBegin = uint32(instances.InstanceCount())
			batch[0].BatchCount = uint32(len(batch))

			for _, item := range batch {
				mesh := &meshes.Meshes[item.ExtractedIndex]

				// write material & store index
				mesh.Material.WriteUniforms(materialUniforms.Writer.Next())
				materialIndex := uint32(materialUniforms.Writer.ItemCount)

				// write sprite vertex data
				instances.StartNew(48)

				// the meshes transform
				instances.AppendVec3f(mesh.Transform.Column(0).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(1).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(2).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(3).Truncate())

				// a reference to the meshes material
				instances.AppendUint(materialIndex)
			}
		}
	}

	// upload buffer to gpu
	instances.WriteTo(ctx, &meshInstances.Buffer, "Mesh2d Instances")
}

func drawMesh2dBatch(world *byke.World, pass *TrackedRenderPassEncoder, item RenderItem) (ok bool) {
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
	meshes *ExtractedMeshes2d,
	instances *mesh2dInstances,
	meshAllocator *MeshAllocator,
	bindGroupCache *MaterialBindGroups,
	viewQuery ViewQuery[struct {
		ViewTarget         *ViewTarget
		ViewUniformsOffset DynamicOffset[ViewUniforms]
	}],
) {
	view := viewQuery.Get()

	pass := task.Value.Pass
	item := task.Value.Item

	mesh := meshes.Meshes[item.ExtractedIndex]

	buf, ok := meshAllocator.Get(mesh.Mesh)
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

	// tell the pipeline about the attributes we want to use
	pipelineConfig.Attributes = append(
		pipelineConfig.Attributes,
		buf.VertexLayout.Attributes...,
	)

	pipeline := pipelines.Specialize(pipelineConfig)

	bindGroup := bindGroupCache.MustLookup(mesh.Material)

	pass.SetPipeline(pipeline.Get())

	pass.SetBindGroup(0, viewBindGroup.BindGroup, []uint32{view.ViewUniformsOffset.Offset})
	pass.SetBindGroup(1, bindGroup, nil)

	pass.SetVertexBuffer(0, instances.Buffer, 0, wgpu.WholeSize)
	pass.SetVertexBuffer(1, buf.Vertices, 0, wgpu.WholeSize)

	pass.SetIndexBuffer(buf.Indices, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
	pass.DrawIndexed(indexCount, item.BatchCount, 0, 0, item.BatchBegin)
}
