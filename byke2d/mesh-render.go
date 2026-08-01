package byke2d

import (
	"bytes"
	"cmp"
	_ "embed"
	"fmt"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[MeshViewBindGroup]()

func pluginMesh3d(app *byke.App) {
	app.InitResource[ExtractedMeshes]()
	app.InitResource[meshInstances]()
	app.InitResource[MeshBindGroups]()
	app.InitResource[skinUniforms]()
	app.InitResource[morphUniforms]()
	app.InitResource[meshPipelineCache]()

	app.AddSystems(byke.Startup, setupBrdfLookupTableSystem)

	app.AddSystems(Render, byke.System(queueMeshInstancesSystem).InSet(RenderPhaseQueue))

	app.AddSystems(Render, byke.System(prepareMeshPipelinesSystems).InSet(RenderPhasePrepare))

	app.AddSystems(Render, byke.System(prepareSkinsUniformsSystem).InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.System(prepareMorphUniformsSystem).
		After(allocateMeshesSystem).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.System(prepareMeshViewBindGroupSystem).InSet(RenderPhasePrepareBindGroups))
	app.AddSystems(Render, byke.System(prepareMeshBindGroupSystem).InSet(RenderPhasePrepareBindGroups))

	app.AddSystems(Render, byke.System(clearExtractedMeshesSystem).InSet(RenderPhaseCleanup))

	// need to sync the Weights to the actual mesh node
	app.AddSystems(PreRender, syncMeshMorphWeightsSystem)

	app.AddPlugin(PluginMaterial[StandardMaterial])
	app.AddPlugin(PluginMaterial[ColorMaterial])

	app.AddPlugin(ComponentUniformsPlugin[EnvironmentMapLight])
}

type ExtractedMeshes struct {
	Meshes []ExtractedMesh
}

func extractMeshesWithMaterialSystem[M Material](
	meshes *ExtractedMeshes,
	materials *Area[M],
	meshQuery byke.Query[struct {
		EntityId        byke.EntityId
		Mesh            query.Ref[MeshInstance]
		Transform       GlobalTransform
		Material        M
		RenderLayers    byke.Option[RenderLayers]
		CustomShader    byke.Option[CustomShader]
		SkinnedMesh     byke.Option[SkinnedMesh]
		HasMorphWeights byke.Has[meshMorphWeights]
		Visibility      ComputedVisibility
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
			Mesh:             mesh.Mesh,
			Transform:        item.Transform.Affine,
			Material:         any(materials.Alloc(item.Material)).(Material),
			RenderLayers:     item.RenderLayers.Or(renderLayerZero),
			Skin:             skin,
			HashMorphWeights: item.HasMorphWeights.Exists(),
			EntityId:         item.EntityId,
		})
	}
}

func clearExtractedMeshesSystem(
	meshes *ExtractedMeshes,
) {
	clear(meshes.Meshes)
	meshes.Meshes = meshes.Meshes[:0]
}

type MeshKey struct {
	MatType   reflect.Type
	MatKey    MaterialBindGroupKey
	LayoutKey VertexLayoutKey
	Mesh      *Mesh
}

func (m MeshKey) CompareTo(other any) int {
	o, ok := other.(MeshKey)
	if !ok {
		return compareByType(m, other)
	}

	return cmp.Or(
		compareType(m.MatType, o.MatType),
		cmp.Compare(m.LayoutKey, o.LayoutKey),
		cmp.Compare(m.MatKey.SortValue(), o.MatKey.SortValue()),
		compareByAddress(m.Mesh, o.Mesh),
	)
}

type meshRenderPhaseItem struct{}

func queueMeshInstancesSystem(
	meshes *ExtractedMeshes,
	meshKeyArea *byke.Local[Area[MeshKey]],
	viewsQuery byke.Query[struct {
		_            byke.With[Camera]
		Transform    GlobalTransform
		RenderLayers RenderLayers
		RenderPhase  *BinnedRenderPhase[Opaque]
		Transparent  *SortableRenderPhase[Transparent]
	}],
) {
	meshKeyArea.Value.Tick()

	for view := range viewsQuery.Items() {
		for idx := range meshes.Meshes {
			sp := &meshes.Meshes[idx]
			if !view.RenderLayers.Intersects(sp.RenderLayers) {
				continue
			}

			renderItem := RenderItem{
				Type:           &meshRenderPhaseItem{},
				Draw:           drawMeshesBatch,
				ExtractedIndex: uint32(idx),
			}

			key := MeshKey{
				MatKey:    sp.Material.BindGroupKey(),
				MatType:   reflect.TypeOf(sp.Material),
				LayoutKey: sp.Mesh.VertexLayout().Key(),
				Mesh:      sp.Mesh,
			}

			if sp.Material.IsOrderIndependent() {
				view.RenderPhase.Append(renderItem, key)

			} else {
				distanceToCameraSq := sp.Transform.Translation().
					Sub(view.Transform.Affine.Translation()).
					LengthSqr()

				// will be sorting ascending, but we want to draw the largest
				// distance first
				distanceToCameraSq = -distanceToCameraSq

				view.Transparent.Append(renderItem, distanceToCameraSq)
			}
		}
	}
}

type meshPipelineCacheKey struct {
	View   byke.EntityId
	Entity byke.EntityId
}

type meshPipelineCache struct {
	tickCache[meshPipelineCacheKey, Pipeline]
}

func prepareMeshPipelinesSystems(
	meshes ExtractedMeshes,
	pipelines *PipelineCache,
	cache *meshPipelineCache,
	viewsQuery byke.Query[struct {
		ViewId              byke.EntityId
		ViewTarget          ViewTarget
		EnvironmentMapLight byke.Has[EnvironmentMapLight]
	}],
) {
	cache.Tick()

	for view := range viewsQuery.Items() {
		for _, mesh := range meshes.Meshes {
			pipelineConfig := meshPipelineConfig{
				Format:       view.ViewTarget.Format,
				SampleCount:  view.ViewTarget.SampleCount,
				Skinned:      mesh.Skin.IsSet(),
				Morph:        mesh.HashMorphWeights,
				VertexLayout: mesh.Mesh.VertexLayout(),
				Material:     mesh.Material,
				MeshView: MeshViewBindGroupLayoutOptions{
					EnvironmentMapLight: view.EnvironmentMapLight.Exists(),
				},
			}

			key := meshPipelineCacheKey{
				View:   view.ViewId,
				Entity: mesh.EntityId,
			}

			pipeline := pipelines.Specialize(pipelineConfig)
			cache.Add(key, pipeline)
		}
	}
}

// meshInstances stores the instance buffer for all per-instance
// data of the meshes
type meshInstances struct {
	Buffer    *wgpu.Buffer
	Instances wgsl.InstanceWriter
}

func prepareMeshInstancesSystem(
	ctx *RenderContext,
	meshes *ExtractedMeshes,
	meshInstances *meshInstances,
	meshAllocator *MeshAllocator,
	morphUniforms *morphUniforms,
	materialUniforms *MaterialUniforms,
	viewsQuery byke.Query[struct {
		_           byke.With[Camera]
		Phase       *BinnedRenderPhase[Opaque]
		Transparent *SortableRenderPhase[Transparent]
	}],
) {
	instances := &meshInstances.Instances
	instances.Clear()

	appendInstance := func(item *ExtractedMesh) {
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

	for view := range viewsQuery.Items() {
		for _, batch := range view.Phase.Batches() {
			if len(batch) == 0 {
				continue
			}

			batch[0].BatchBegin = uint32(instances.InstanceCount())
			batch[0].BatchCount = uint32(len(batch))

			for _, item := range batch {
				item := &meshes.Meshes[item.ExtractedIndex]
				appendInstance(item)
			}
		}

		for idx := range view.Transparent.Len() {
			item := view.Transparent.Get(idx)

			if _, isMesh := item.Type.(*meshRenderPhaseItem); !isMesh {
				continue
			}

			item.BatchBegin = uint32(instances.InstanceCount())
			item.BatchCount = uint32(1)

			mesh := &meshes.Meshes[item.ExtractedIndex]
			appendInstance(mesh)
		}
	}

	// upload buffer to gpu
	instances.WriteTo(ctx, &meshInstances.Buffer, "Mesh Instances")
}

type MeshViewBindGroupLayoutOptions struct {
	EnvironmentMapLight bool
}

func MeshViewBindGroupLayout(opts MeshViewBindGroupLayoutOptions) wgpu.BindGroupLayoutDescriptor {
	var bindings = []wgpu.BindGroupLayoutEntry{
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
	}

	if opts.EnvironmentMapLight {
		bindings = append(bindings,
			// For PBR rendering, we might need the environment map & specular lighting
			// as well as its configuration parameters. If we do not have an environment map,
			// we'll bind an empty texture
			Indexed(40, BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true)),
			Indexed(41, BindingLayoutTextureCube(wgpu.TextureSampleTypeFloat, false)),
			Indexed(42, BindingLayoutTextureCube(wgpu.TextureSampleTypeFloat, false)),
			Indexed(43, BindingLayoutTexture2D(wgpu.TextureSampleTypeFloat, false)),
			Indexed(44, BindingLayoutSampler(wgpu.SamplerBindingTypeFiltering)),
		)
	}

	return SequentialLayoutWithLabel("MeshView", bindings...)
}

type MeshViewBindGroup struct {
	byke.Component[MeshViewBindGroup]
	Options   MeshViewBindGroupLayoutOptions
	BindGroup *wgpu.BindGroup
}

// prepareMeshViewBindGroupSystem prepares the "view" bind group for the mesh pipeline.
// It contains references to the projection, lights, etc.
// This is created once per view.
func prepareMeshViewBindGroupSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	viewBindGroup ViewBindGroup,
	morphUniforms morphUniforms,
	skinUniforms skinUniforms,
	lights *lightsStorage,
	viewUniforms *ComponentUniforms[ViewUniforms],
	environments *ComponentUniforms[EnvironmentMapLight],
	brdfLookupTable brdfLookupTable,
	cameraQuery byke.Query[struct {
		_                   byke.With[ViewTarget]
		EntityId            byke.EntityId
		EnvironmentMapLight byke.Option[EnvironmentMapLight]
		BindGroup           byke.OptionMut[MeshViewBindGroup]
	}],
) {
	for camera := range cameraQuery.Items() {
		var opts MeshViewBindGroupLayoutOptions

		entries := []wgpu.BindGroupEntry{
			Indexed(0, viewUniforms.Binding()),
			Indexed(1, BindingBuffer(viewBindGroup.BufferGlobals)),

			Indexed(10, BindingBuffer(lights.BufConfig)),
			Indexed(11, BindingBuffer(lights.BufDirectionalLights)),
			Indexed(12, BindingBuffer(lights.BufPointLights)),
			Indexed(13, BindingBuffer(lights.BufSpotLights)),

			Indexed(20, BindingBuffer(morphUniforms.BufDescriptors)),
			Indexed(21, BindingBuffer(morphUniforms.BufWeights)),

			Indexed(30, BindingBufferSize(skinUniforms.BufJoints, 0, 64*256)),
		}

		if env, ok := camera.EnvironmentMapLight.Get(); ok {
			opts.EnvironmentMapLight = true

			entries = append(entries,
				Indexed(40, environments.Binding()),
				Indexed(41, BindingTextureView(env.DiffuseMap.TextureView)),
				Indexed(42, BindingTextureView(env.SpecularMap.TextureView)),
				Indexed(43, BindingTextureView(brdfLookupTable.Texture.TextureView)),
				Indexed(44, BindingSampler(brdfLookupTable.Texture.Sampler)),
			)
		}

		layout := MeshViewBindGroupLayout(opts)

		bindGroup := MeshViewBindGroup{
			Options: opts,
			BindGroup: ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:   "MeshView",
				Layout:  ctx.CreateBindGroupLayout(layout),
				Entries: Sequential(entries...),
			}),
		}

		if b, ok := camera.BindGroup.Get(); ok {
			// release previous bind group
			b.BindGroup.Release()

			// and replace with new one
			*b = bindGroup
		} else {
			commands.Entity(camera.EntityId).Insert(bindGroup)
		}
	}
}

type brdfLookupTable struct {
	Texture *Texture
}

//go:embed brdf-lut.ktx2
var brdfLookupTableTextureKTX []byte

func setupBrdfLookupTableSystem(commands *byke.Commands, ctx *RenderContext) {
	tex, err := LoadKTXToTexture(ctx,
		bytes.NewReader(brdfLookupTableTextureKTX),
		LoadKTXTextureOptions{},
		"brdf lut")

	if err != nil {
		panic(fmt.Errorf("load brdf-lut: %w", err))
	}

	commands.InsertResource(brdfLookupTable{Texture: tex})
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
	meshes *ExtractedMeshes,
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

var drawMeshesBatchSystemCached = byke.AsCachedSystem(drawMeshesBatchSystem)

func drawMeshesBatch(world *byke.World, pass *TrackedRenderPassEncoder, item RenderItem) (ok bool) {
	world.RunSystemWithInValue(drawMeshesBatchSystemCached, RenderTask{
		Pass: pass,
		Item: item,
	})

	return true
}

func drawMeshesBatchSystem(
	task byke.In[RenderTask],
	meshBindGroups MeshBindGroups,
	meshes *ExtractedMeshes,
	meshInstances *meshInstances,
	meshAllocator *MeshAllocator,
	meshPipelineCache *meshPipelineCache,
	materialBindGroups *MaterialBindGroups,
	skinUniforms *skinUniforms,
	viewQuery ViewQuery[struct {
		ViewId              byke.EntityId
		ViewTarget          *ViewTarget
		MeshBindGroup       MeshViewBindGroup
		ViewUniformsOffset  DynamicOffset[ViewUniforms]
		EnvironmentMapLight byke.Option[DynamicOffset[EnvironmentMapLight]]
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

	skinOffset, _ := skinUniforms.OffsetOf(mesh.Skin.EntityId)

	pipeline, ok := meshPipelineCache.Get(meshPipelineCacheKey{
		View:   view.ViewId,
		Entity: mesh.EntityId,
	})
	if !ok {
		panic("mesh pipeline not found")
	}

	materialBindGroup := materialBindGroups.MustLookup(mesh.Material)

	meshBindGroup, ok := meshBindGroups.ByMesh(mesh.Mesh)
	if !ok {
		panic("mesh bind group is missing")
	}

	dynamicOffsetsForViewGroup := []uint32{
		view.ViewUniformsOffset.Offset,
		skinOffset,
	}

	if off, ok := view.EnvironmentMapLight.Get(); ok {
		dynamicOffsetsForViewGroup = append(dynamicOffsetsForViewGroup, off.Offset)
	}

	pass.SetPipeline(pipeline.Get())

	pass.SetBindGroup(0, view.MeshBindGroup.BindGroup, dynamicOffsetsForViewGroup)
	pass.SetBindGroup(1, meshBindGroup, nil)
	pass.SetBindGroup(2, materialBindGroup, nil)

	// per instance data, like transformation, indices in global buffers, etc
	pass.SetVertexBuffer(0, meshInstances.Buffer, 0, wgpu.WholeSize)

	// the per vertex data for the current mesh
	pass.SetVertexBuffer(1, buf.Vertices, 0, wgpu.WholeSize)

	pass.SetIndexBuffer(buf.Indices, buf.IndexFormat, 0, wgpu.WholeSize)

	pass.DrawIndexed(buf.IndicesCount, item.BatchCount, buf.FirstIndex, int32(buf.FirstVertex), item.BatchBegin)
}
