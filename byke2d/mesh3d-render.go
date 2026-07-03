package byke2d

import (
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh3d(app *byke.App) {
	app.InsertResource(ExtractedMesh3d{})
	app.InsertResource(mesh3dInstances{})
	app.InsertResource(MeshBindGroups{})
	app.InsertResource(skinUniforms{})
	app.InsertResource(morphUniforms{})

	app.AddSystems(Render, byke.System(queueMesh3dSystem).InSet(RenderPhaseQueue))

	app.AddSystems(Render, byke.System(prepareSkinsUniformsSystem).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareMorphUniformsSystem).InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.System(prepareMeshViewBindGroupSystem).InSet(RenderPhasePrepareBindGroups))
	app.AddSystems(Render, byke.System(prepareMeshBindGroupSystem).InSet(RenderPhasePrepareBindGroups))

	app.AddSystems(Render, byke.System(clearExtractedMesh3dSystem).InSet(RenderPhaseCleanup))

	// need to sync the Weights to the actual mesh node
	app.AddSystems(PreRender, syncMeshMorphWeightsSystem)

	app.AddPlugin(PluginMaterial3d[StandardMaterial])
}

type ExtractedMesh3d struct {
	Meshes []ExtractedMesh
}

func extractMesh3dSystem[M Material](
	meshes *ExtractedMesh3d,
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
	meshes *ExtractedMesh3d,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type mesh3dRenderPhaseItem struct{}

var drawMesh3dBatchStandardMaterial = drawMesh3dBatch[StandardMaterial]()

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
				Draw:           drawMesh3dBatchStandardMaterial,
				ExtractedIndex: uint32(idx),
			}

			key := MeshKey{
				Type:    &mesh3dRenderPhaseItem{},
				MatKey:  sp.Material.Key(),
				MatType: reflect.TypeOf(sp.Material),
				Mesh:    sp.Mesh,
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

func prepareMesh3dInstances[M Material](
	ctx *RenderContext,
	meshes *ExtractedMesh3d,
	meshInstances *mesh3dInstances,
	morphUniforms *morphUniforms,
	materialUniforms *MaterialUniforms[M],
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

			if key.MatType != reflect.TypeFor[M]() {
				// wrong material
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
				instances.StartNew(56)

				// transform
				instances.AppendVec3f(mesh.Transform.Column(0).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(1).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(2).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(3).Truncate())

				// material index
				instances.AppendUint(materialIndex)

				// reference morph info if available
				idx, _ := morphUniforms.DescriptorIndex(mesh.EntityId)
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
	groups   tickCache[*Mesh, *wgpu.BindGroup]
	emptyBuf *wgpu.Buffer
}

func (m *MeshBindGroups) ByMesh(mesh *Mesh) (*wgpu.BindGroup, bool) {
	return m.groups.Get(mesh)
}

var MeshBindGroupLayout = SequentialLayoutWithLabel("Mesh",
	// morph attributes
	BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false),
)

func prepareMeshBindGroupSystem(
	ctx *RenderContext,
	bindGroups *MeshBindGroups,
	meshes *ExtractedMesh3d,
	meshAllocator *MeshAllocator,
) {
	if bindGroups.emptyBuf == nil {
		bindGroups.emptyBuf = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "empty",
			Contents: []byte{0, 0, 0, 0},
			Usage:    wgpu.BufferUsageStorage | wgpu.BufferUsageUniform,
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

		_ = buf
		// TODO fix morph targets here
		var morphAttributes *wgpu.Buffer = nil // := buf.MorphAttributes
		if morphAttributes == nil {
			morphAttributes = bindGroups.emptyBuf
		}

		// create and cache new bind group for this mesh
		bindGroups.groups.Add(mesh.Mesh, ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Mesh",
			Layout: ctx.CreateBindGroupLayout(MeshBindGroupLayout),
			Entries: Sequential(
				BindingBuffer(morphAttributes),
			),
		}))
	}
}

func drawMesh3dBatch[M Material]() Draw {
	var drawSystem = drawMesh3dBatchSystem[M]

	return func(world *byke.World, pass *TrackedRenderPassEncoder, item RenderItem) (ok bool) {
		world.RunSystemWithInValue(drawSystem, RenderTask{
			Pass: pass,
			Item: item,
		})

		return true
	}
}

func drawMesh3dBatchSystem[M Material](
	task byke.In[RenderTask],
	meshViewBindGroup meshViewBindGroup,
	meshBindGroups MeshBindGroups,
	pipelines *PipelineCache,
	meshes *ExtractedMesh3d,
	instances *mesh3dInstances,
	meshAllocator *MeshAllocator,
	materialBindGroups *MaterialBindGroups[M],
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

	var layout []wgpu.BindGroupLayoutEntry
	layout = append(layout, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false))
	layout = append(layout, mesh.Material.BindingsLayout()...)

	pipelineConfig := mesh3dPipelineConfig{
		Format:           view.ViewTarget.Format,
		SampleCount:      view.ViewTarget.SampleCount,
		Shader:           mesh.Material.Shader(),
		MaterialBindings: layout,
		Skinned:          skinOk && mesh.Skin.IsSet(),
		Morph:            morphOk,
		VertexLayout:     buf.VertexLayout,
	}

	pipeline := pipelines.Specialize(pipelineConfig)

	materialBindGroup, _ := materialBindGroups.Get(mesh.Material.Key())
	meshBindGroup, ok := meshBindGroups.ByMesh(mesh.Mesh)
	if !ok {
		panic("mesh bind group is missing")
	}

	pass.SetPipeline(pipeline.Get())

	pass.SetBindGroup(0, meshViewBindGroup.BindGroup, []uint32{view.ViewUniformsOffset.Offset, skinOffset})
	pass.SetBindGroup(1, meshBindGroup, nil)
	pass.SetBindGroup(2, materialBindGroup, nil)

	// per instance data, like transformation, indices in global buffers, etc
	pass.SetVertexBuffer(0, instances.Buffer, 0, wgpu.WholeSize)

	// the per vertex data for the current mesh
	pass.SetVertexBuffer(1, buf.Vertices, 0, wgpu.WholeSize)

	pass.SetIndexBuffer(buf.Indices, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
	pass.DrawIndexed(buf.IndicesCount, item.BatchCount, buf.FirstIndex, int32(buf.FirstVertex), item.BatchBegin)
}
