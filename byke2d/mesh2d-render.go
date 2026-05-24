package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func pluginMesh2d(app *byke.App) {
	app.InsertResource(ExtractedMeshes{})
	app.InsertResource(meshInstances{})
	app.InsertResource(materialBindGroupCache{})

	app.InsertResource(byke.InitFromWorld(mesh2dCacheFromWorld))

	app.AddSystems(Render, byke.System(queueMeshesSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(prepareMesh2dBuffers).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareMesh2dInstances).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(clearExtractedMeshesSystem).InSet(RenderPhaseCleanup))

	app.AddPlugin(PluginMaterial2d[ColorMaterial])
}

type ComparableMaterial interface {
	comparable
	Material
}

func PluginMaterial2d[M ComparableMaterial](app *byke.App) {
	app.AddSystems(Render, byke.System(extractMeshesSystem[M]).InSet(RenderPhaseExtract))
}

type materialBindGroupCache struct {
	tickCache[Material, *wgpu.BindGroup]
}

type ExtractedMesh struct {
	Mesh *Mesh

	Transform    glm.Mat4f
	RenderLayers RenderLayers
	Material     Material
}

type ExtractedMeshes struct {
	Meshes []ExtractedMesh
}

func extractMeshesSystem[M Material](
	meshes *ExtractedMeshes,
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
	meshes byke.Query[*Mesh2d],
	meshCache *mesh2dCache,
) {
	meshCache.Reset()

	for item := range meshes.Items() {
		mesh := item.Mesh
		forceUpload := mesh.requireUpload()
		meshCache.Upload(mesh, forceUpload)
		mesh.markUploaded()
	}
}

// meshInstances stores the instance buffer for all per-instance
// data of the meshes
type meshInstances struct {
	Buffer    *wgpu.Buffer
	Instances wgsl.InstanceWriter
}

func prepareMesh2dInstances(
	ctx *RenderContext,
	meshes *ExtractedMeshes,
	pipelineCache *PipelineCache,
	meshInstances *meshInstances,
	bindGroups *materialBindGroupCache,
	viewsQuery byke.Query[struct {
		_     byke.With[Camera]
		Phase RenderPhase
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

func createMaterialBindGroup(ctx *RenderContext, pipelines *PipelineCache, material Material) *wgpu.BindGroup {
	// TODO create buffer each time is not good, must be saved & reused somewhere.
	//  we probably need to store BindGroup together with the buffer
	var w wgsl.StructWriter
	material.WriteUniforms(&w)

	label := fmt.Sprintf("material group: %T", material)

	buffer := ctx.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    label,
		Usage:    wgpu.BufferUsageUniform,
		Contents: w.Bytes(),
	})

	var bindings []wgpu.BindGroupEntry
	bindings = append(bindings, BindingBuffer(buffer))
	bindings = append(bindings, material.Bindings()...)

	var layout []wgpu.BindGroupLayoutEntry
	layout = append(layout, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, false))
	layout = append(layout, material.BindingsLayout()...)

	return ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   label,
		Layout:  pipelines.BindGroupLayout(SequentialLayout(layout...)),
		Entries: Sequential(bindings...),
	})
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
	pipelines *PipelineCache,
	task byke.In[RenderTask],
	meshes *ExtractedMeshes,
	instances *meshInstances,
	meshCache *mesh2dCache,
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
