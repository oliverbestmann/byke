package byke2d

import (
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh3d(app *byke.App) {
	app.InsertResource(ExtractedMeshes3d{})
	app.InsertResource(mesh3dInstances{})
	app.InsertResource(MeshBindGroups{})
	app.InsertResource(skinUniforms{})
	app.InsertResource(morphUniforms{})

	app.AddSystems(Render, byke.System(queueMesh3dSystem).InSet(RenderPhaseQueue))

	app.AddSystems(Render, byke.System(prepareSkinsUniformsSystem).InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.System(prepareMorphUniformsSystem).
		After(prepareMesh3dBuffers).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.System(prepareMeshViewBindGroupSystem).InSet(RenderPhasePrepareBindGroups))
	app.AddSystems(Render, byke.System(prepareMeshBindGroupSystem).InSet(RenderPhasePrepareBindGroups))

	app.AddSystems(Render, byke.System(clearExtractedMesh3dSystem).InSet(RenderPhaseCleanup))

	// need to sync the Weights to the actual mesh node
	app.AddSystems(PreRender, syncMeshMorphWeightsSystem)

	app.AddPlugin(PluginMaterial3d[StandardMaterial])
}

type ExtractedMeshes3d struct {
	Meshes []ExtractedMesh
}

func extractMesh3dSystem[M Material](
	meshes *ExtractedMeshes3d,
	meshQuery byke.Query[struct {
		EntityId     byke.EntityId
		Mesh         query.Ref[Mesh3d]
		Transform    GlobalTransform
		Material     M
		RenderLayers byke.Option[RenderLayers]
		CustomShader byke.Option[CustomShader]
		SkinnedMesh  byke.Option[SkinnedMesh]
		Visibility   ComputedVisibility
	}],
) {
	for item := range meshQuery.Items() {
		if !item.Visibility.Visible {
			continue
		}

		mesh := item.Mesh.Value

		var skin ExtractedSkin
		if sm, ok := item.SkinnedMesh.Get(); ok {
			skin.EntityId = item.EntityId
			skin.Joints = sm.Joints
			skin.InverseBind = sm.InverseBind
		}

		meshes.Meshes = append(meshes.Meshes, ExtractedMesh{
			Mesh:         mesh.Mesh,
			Transform:    item.Transform.Affine,
			Material:     item.Material,
			RenderLayers: item.RenderLayers.Or(renderLayerZero),
			Skin:         skin,
			EntityId:     item.EntityId,
		})
	}
}

func clearExtractedMesh3dSystem(
	meshes *ExtractedMeshes3d,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type mesh3dRenderPhaseItem struct{}

func queueMesh3dSystem(
	meshes *ExtractedMeshes3d,
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

			key := &MeshKey{
				Type:      &mesh3dRenderPhaseItem{},
				MatKey:    sp.Material.BindGroupKey(),
				MatType:   reflect.TypeOf(sp.Material),
				LayoutKey: sp.Mesh.VertexLayout().Key(),
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

func prepareMesh3dInstancesSystem(
	ctx *RenderContext,
	meshes *ExtractedMeshes3d,
	meshInstances *mesh3dInstances,
	meshAllocator *MeshAllocator,
	morphUniforms *morphUniforms,
	materialUniforms *MaterialUniforms,
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

			_, isMesh := key.Type.(*mesh3dRenderPhaseItem)

			if !isMesh {
				// this batch is not a mesh
				continue
			}

			batch[0].BatchBegin = uint32(instances.InstanceCount())
			batch[0].BatchCount = uint32(len(batch))

			for _, item := range batch {
				item := &meshes.Meshes[item.ExtractedIndex]

				bufs, ok := meshAllocator.Get(item.Mesh)
				if !ok {
					panic("mesh not found")
				}

				// write mesh instance data
				instances.StartNew(60)

				// transform
				instances.AppendVec3f(item.Transform.Column(0).Truncate())
				instances.AppendVec3f(item.Transform.Column(1).Truncate())
				instances.AppendVec3f(item.Transform.Column(2).Truncate())
				instances.AppendVec3f(item.Transform.Column(3).Truncate())

				// initial vertex position
				instances.AppendUint(bufs.FirstVertex)

				// material index
				instances.AppendUint(materialUniforms.Get(item.Material).Indices[item.EntityId])

				// reference morph info if available
				idx, _ := morphUniforms.DescriptorIndex(item.EntityId)
				instances.AppendUint(idx)
			}
		}
	}

	// upload buffer to gpu
	instances.WriteTo(ctx, &meshInstances.Buffer, "Mesh3d Instances")
}

var MeshViewBindGroupLayout = SequentialLayout(
	// View, offset by active ViewUniforms
	Indexed(0, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true)),

	// Globals
	Indexed(1, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false)),

	// All the lights
	Indexed(10, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false)),
	Indexed(11, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false)),
	Indexed(12, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false)),
	Indexed(13, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false)),

	// All morph descriptors
	Indexed(20, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false)),

	// All morph weights
	Indexed(21, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false)),

	// All skin joint transforms, offset by entityId
	Indexed(30, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true)),
)

type meshViewBindGroup struct {
	BindGroup *wgpu.BindGroup
}

func prepareMeshViewBindGroupSystem(
	ctx *RenderContext,
	bindGroup *meshViewBindGroup,
	viewBindGroup ViewBindGroup,
	morphUniforms morphUniforms,
	skinUniforms skinUniforms,
	lights *lightsStorage,
	viewUniforms *ComponentUniforms[ViewUniforms],
) {
	bindGroup.BindGroup.Release()

	bindGroup.BindGroup = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "MeshView",
		Layout: ctx.CreateBindGroupLayout(MeshViewBindGroupLayout),
		Entries: Sequential(
			Indexed(0, viewUniforms.Binding()),
			Indexed(1, BindingBuffer(viewBindGroup.BufferGlobals)),

			Indexed(10, BindingBuffer(lights.BufConfig)),
			Indexed(11, BindingBuffer(lights.BufDirectionalLights)),
			Indexed(12, BindingBuffer(lights.BufPointLights)),
			Indexed(13, BindingBuffer(lights.BufSpotLights)),

			Indexed(20, BindingBuffer(morphUniforms.BufDescriptors)),
			Indexed(21, BindingBuffer(morphUniforms.BufWeights)),

			Indexed(30, BindingBufferSize(skinUniforms.BufJoints, 0, 64*256)),
		),
	})
}

// MeshBindGroups holds the per mesh bind group containing mesh
// specific data, such as the morph attribute data
type MeshBindGroups struct {
	// has dynamic offset configured for the start of the joints array
	groups         tickCache[*Mesh, *wgpu.BindGroup]
	emptyBindGroup *wgpu.BindGroup
}

func (m *MeshBindGroups) ByMesh(mesh *Mesh) (*wgpu.BindGroup, bool) {
	return m.groups.Get(mesh)
}

var MeshBindGroupLayout = SequentialLayoutWithLabel(
	"Mesh",
	// morph attributes
	BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false),
)

func prepareMeshBindGroupSystem(
	ctx *RenderContext,
	bindGroups *MeshBindGroups,
	meshes *ExtractedMeshes3d,
	meshAllocator *MeshAllocator,
) {
	if bindGroups.emptyBindGroup == nil {
		emptyBuf := ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "empty",
			Contents: []byte{0, 0, 0, 0},
			Usage:    wgpu.BufferUsageStorage | wgpu.BufferUsageUniform,
		})

		bindGroups.emptyBindGroup = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Mesh",
			Layout: ctx.CreateBindGroupLayout(MeshBindGroupLayout),
			Entries: Sequential(
				BindingBuffer(emptyBuf),
			),
		})
	}

	for _, mesh := range meshes.Meshes {
		// TODO check for change in morph attributes buffer
		if _, ok := bindGroups.groups.Get(mesh.Mesh); ok {
			continue
		}

		buf, ok := meshAllocator.Get(mesh.Mesh)
		if !ok {
			continue
		}

		if buf.MorphAttributes == nil {
			bindGroups.groups.Add(mesh.Mesh, bindGroups.emptyBindGroup)
			continue
		}

		// create and cache new bind group for this mesh
		bindGroups.groups.Add(mesh.Mesh, ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Mesh",
			Layout: ctx.CreateBindGroupLayout(MeshBindGroupLayout),
			Entries: Sequential(
				BindingBuffer(buf.MorphAttributes),
			),
		}))
	}
}

func drawMesh3dBatch(world *byke.World, pass *TrackedRenderPassEncoder, item RenderItem) (ok bool) {
	world.RunSystemWithInValue(drawMesh3dBatchSystem, RenderTask{
		Pass: pass,
		Item: item,
	})

	return true
}

func drawMesh3dBatchSystem(
	task byke.In[RenderTask],
	meshViewBindGroup meshViewBindGroup,
	meshBindGroups MeshBindGroups,
	pipelines *PipelineCache,
	meshes *ExtractedMeshes3d,
	meshInstances *mesh3dInstances,
	meshAllocator *MeshAllocator,
	materialBindGroups *MaterialBindGroups,
	skinUniforms *skinUniforms,
	morphUniforms *morphUniforms,
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

	skinOffset, skinOk := skinUniforms.OffsetOf(mesh.Skin.EntityId)
	_, morphOk := morphUniforms.DescriptorIndex(mesh.EntityId)

	pipelineConfig := mesh3dPipelineConfig{
		Format:       view.ViewTarget.Format,
		SampleCount:  view.ViewTarget.SampleCount,
		Skinned:      skinOk && mesh.Skin.IsSet(),
		Morph:        morphOk,
		VertexLayout: mesh.Mesh.VertexLayout(),
		Material:     mesh.Material,
	}

	pipeline := pipelines.Specialize(pipelineConfig)

	materialBindGroup := materialBindGroups.MustLookup(mesh.Material)

	meshBindGroup, ok := meshBindGroups.ByMesh(mesh.Mesh)
	if !ok {
		panic("mesh bind group is missing")
	}

	pass.SetPipeline(pipeline.Get())

	pass.SetBindGroup(0, meshViewBindGroup.BindGroup, []uint32{view.ViewUniformsOffset.Offset, skinOffset})
	pass.SetBindGroup(1, meshBindGroup, nil)
	pass.SetBindGroup(2, materialBindGroup, nil)

	// per instance data, like transformation, indices in global buffers, etc
	pass.SetVertexBuffer(0, meshInstances.Buffer, 0, wgpu.WholeSize)

	// the per vertex data for the current mesh
	pass.SetVertexBuffer(1, buf.Vertices, 0, wgpu.WholeSize)

	pass.SetIndexBuffer(buf.Indices, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
	pass.DrawIndexed(buf.IndicesCount, item.BatchCount, buf.FirstIndex, int32(buf.FirstVertex), item.BatchBegin)
}
