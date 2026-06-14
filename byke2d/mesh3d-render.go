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
	app.InsertResource(MeshBindGroups{})
	app.InsertResource(skinUniforms{})
	app.InsertResource(morphUniforms{})

	app.AddSystems(Render, byke.System(queueMesh3dSystem).InSet(RenderPhaseQueue))

	app.AddSystems(Render, byke.System(prepareSkinsUniformsSystem).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareMorphUniformsSystem).InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.System(prepareMeshViewBindGroupSystem).InSet(RenderPhasePrepareBindGroups))
	app.AddSystems(Render, byke.System(prepareMeshBindGroupSystem).InSet(RenderPhasePrepareBindGroups))
	app.AddSystems(Render, byke.System(prepareMesh3dInstances).InSet(RenderPhasePrepareBindGroups))

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
	morphUniforms *morphUniforms,
	bindGroups *MaterialBindGroups,
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
				instances.StartNew(52)

				// transform
				instances.AppendVec3f(mesh.Transform.Column(0).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(1).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(2).Truncate())
				instances.AppendVec3f(mesh.Transform.Column(3).Truncate())

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

	// TODO All directed lights
	// Indexed(10, ...),

	// All point lights
	Indexed(11, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false)),

	// TODO All spot lights lights
	// Indexed(11, ...),

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
	pipelines *PipelineCache,
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
		Layout: pipelines.BindGroupLayout(MeshViewBindGroupLayout),
		Entries: Sequential(
			Indexed(0, viewUniforms.Binding()),
			Indexed(1, BindingBuffer(viewBindGroup.BufferGlobals)),

			Indexed(11, BindingBuffer(lights.Buffer)),

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
	buffers *meshCache,
	pipelines *PipelineCache,
) {
	if bindGroups.emptyBuf == nil {
		bindGroups.emptyBuf = ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "empty",
			Contents: []byte{0, 0, 0, 0},
			Usage:    wgpu.BufferUsageStorage | wgpu.BufferUsageUniform,
		})
	}

	buffers.cache.Tick()

	for _, mesh := range meshes.Meshes {
		// TODO check for change in morph attributes buffer
		if _, ok := bindGroups.groups.Get(mesh.Mesh); ok {
			continue
		}

		buf, ok := buffers.Get(mesh.Mesh)
		if !ok {
			continue
		}

		morphAttributes := buf.MorphAttributes
		if morphAttributes == nil {
			morphAttributes = bindGroups.emptyBuf
		}

		// create and cache new bind group for this mesh
		bindGroups.groups.Add(mesh.Mesh, ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Label:  "Mesh",
			Layout: pipelines.BindGroupLayout(MeshBindGroupLayout),
			Entries: Sequential(
				BindingBuffer(morphAttributes),
			),
		}))
	}
}

func drawMesh3dBatch(world *byke.World, pass *wgpu.RenderPassEncoder, item RenderItem) (ok bool) {
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
	meshes *ExtractedMesh3d,
	instances *mesh3dInstances,
	meshCache *meshCache,
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

	buf, ok := meshCache.Get(mesh.Mesh)
	if !ok {
		// mesh not in cache, broken?
		panic("mesh data not in cache")
	}

	indexCount := uint32(len(mesh.Mesh.indices))

	skinOffset, skinOk := skinUniforms.OffsetOf(mesh.Skin.EntityId)
	_, morphOk := morphUniforms.DescriptorIndex(mesh.EntityId)

	var layout []wgpu.BindGroupLayoutEntry
	layout = append(layout, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false))
	layout = append(layout, mesh.Material.BindingsLayout()...)

	pipelineConfig := mesh3dPipelineConfig{
		Format:           view.ViewTarget.Format,
		SampleCount:      view.ViewTarget.SampleCount,
		Shader:           mesh.Material.Shader(),
		MaterialBindings: layout,
		Skinned:          skinOk && mesh.Skin.IsSet(),
		Morph:            morphOk,
	}

	for idx := range buf.Attributes {
		// tell the pipeline about the attributes we want to use
		pipelineConfig.Attributes = append(
			pipelineConfig.Attributes,
			buf.Attributes[idx].Attribute,
		)
	}

	pipeline := pipelines.Specialize(pipelineConfig)

	materialBindGroup, _ := materialBindGroups.Get(mesh.Material)
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

	// the position vertex data for the current mesh
	pass.SetVertexBuffer(1, buf.Vertex, 0, wgpu.WholeSize)

	// TODO pack vertex data per mesh into a single buffer
	// set vertex buffers for other attributes
	for idx := range buf.Attributes {
		buffer := buf.Attributes[idx].Buffer
		pass.SetVertexBuffer(uint32(2+idx), buffer, 0, wgpu.WholeSize)
	}

	pass.SetIndexBuffer(buf.Indices, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
	pass.DrawIndexed(indexCount, item.BatchCount, 0, 0, item.BatchBegin)
}
