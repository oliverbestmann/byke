package byke2d

import (
	"fmt"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/internal/query"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Material defines a abstract material in our renderer.
//
// A material is split into three components:
//
//   - One is the actual material values that can only change by changing a bind group.
//     This includes the textures of the material.
//
//   - One is the data that can be written into a storage buffer and that can be
//     accessed via a per instance index. We have one buffer per material type.
//
//   - A description how to render. This includes the Shader and its bind group layout,
//     as well as the Bindings itself.
type Material interface {
	// WriteUniforms write the non bind group related material data to a struct.
	// This might be the tint of a color material.
	WriteUniforms(w *wgsl.StructWriter)

	// BindGroup returns the MaterialBindGroupHandle for this material. The returned value
	// should be a cached handle from the MaterialBindGroupHandleCache.
	BindGroup() *MaterialBindGroupHandle
}

// ResolvableMaterial must be implemented on a material with pointer receiver,
// as it is supposed to set the material group handle.
type ResolvableMaterial interface {
	ResolveBindGroup(cache *MaterialBindGroupHandleCache)
}

type MaterialBindGroupId uint64

type MaterialBindGroupHandle struct {
	// monotonic, stable sort key
	Id MaterialBindGroupId

	// canonical MaterialBindGroup instance (a pointer)
	BindGroup MaterialBindGroup

	// PipelineKey is computed once at intern time
	PipelineKey MaterialPipelineKey

	// OrderIndependent can also be computed once at insert time
	OrderIndependent bool
}

type MaterialBindGroup interface {
	// Shader returns the shader for the material in its current configuration
	Shader() *ShaderDef

	// Bindings return the Bindings that are to be passed to the pipeline.
	Bindings() []wgpu.BindGroupEntry

	// PipelineKey returns key that indicates that the bind group layout or the pipeline
	// specialization for this material is different.
	PipelineKey() MaterialPipelineKey

	// Specialize specializes the provided pipeline.
	Specialize(pipeline *RenderPipelineDescriptor)

	// IsOrderIndependent indicates that this material can be drawn in arbitrary order
	// with other order independent materials.
	IsOrderIndependent() bool
}

type MaterialValues struct {
	// FrontFace defaults to wgpu.FrontFaceCCW
	FrontFace wgpu.FrontFace

	// AlphaMode decides on the way this material handles alpha values.
	AlphaMode AlphaMode

	// DoubleSided enables double-sided lighting.
	// Need to flip the backface vertex in pixel shader
	DoubleSided bool
}

// IsOrderIndependent mirrors Material.OrderIndependent
func (m MaterialValues) IsOrderIndependent() bool {
	switch m.AlphaMode {
	case AlphaModeOpaque, AlphaModeMask, AlphaModeAlphaToCoverage:
		return true

	case AlphaModeBlend, Premultiplied, AlphaModeAdd, AlphaModeMultiply:
		return false

	default:
		panic(fmt.Errorf("unknown alpha mode %d", m.AlphaMode))
	}
}

func (m MaterialValues) Specialize(pipeline *RenderPipelineDescriptor) {
	pipeline.Primitive.FrontFace = frontFaceOf(m.FrontFace)

	if m.DoubleSided {
		// disable culling so we can render both sides of the triangles
		pipeline.Primitive.CullMode = wgpu.CullModeNone
	}

	switch m.AlphaMode {
	case AlphaModeAlphaToCoverage:
		pipeline.Multisample.AlphaToCoverageEnabled = true

	case AlphaModeBlend:
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStateAlphaBlending
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse

	case Premultiplied:
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStatePremultipliedAlphaBlending
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse

	case AlphaModeAdd:
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStateAdd
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse

	case AlphaModeMultiply:
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStateMultiply
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse

	default:
		// no specialization needed
	}
}

func (m MaterialValues) BindGroupKey() Hash {
	var hash Hash = 0xdead
	hash.Int(m.FrontFace)
	hash.Int(m.AlphaMode)
	return hash
}

func (m MaterialValues) PipelineKey() Hash {
	var hash Hash = 0xdeadbeef
	hash.Int(m.FrontFace)
	hash.Int(m.AlphaMode)
	hash.Bool(m.DoubleSided)
	return hash
}

type MaterialBindGroupKey uint64

func (k MaterialBindGroupKey) SortValue() uint64 {
	return uint64(k)
}

type MaterialPipelineKey uint64

func (k MaterialPipelineKey) SortValue() uint64 {
	return uint64(k)
}

func pluginMaterialCommon(app *byke.App) {
	app.InsertResource(MaterialBindGroups{})
	app.InsertResource(MaterialBindGroupHandleCache{})
	app.InsertResource(MaterialUniforms{})

	app.AddSystems(PreRender, tickMaterialBindGroupsSystems)

	app.AddSystems(Render, byke.
		System(prepareMaterialUniforms).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMeshInstancesSystem).
		After(prepareMaterialUniforms).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMaterialBindGroupsSystem).
		InSet(RenderPhasePrepareBindGroups))
}

func PluginMaterial[M isMaterialComponent[M], PM interface {
	*M
	isMaterialComponent[M]
}](app *byke.App) {
	app.InitResource[Area[M]]()

	app.AddSystems(PreRender, byke.
		System(tickMaterialAreaSystem[M]))

	app.AddSystems(PreRender, byke.
		System(resolveMaterialBindGroupsSystem[M, PM]))

	app.AddSystems(Render, byke.
		System(extractMeshesWithMaterialSystem[M]).
		InSet(RenderPhaseExtract))

}

type MaterialUniforms struct {
	// by material type
	byMaterial map[reflect.Type]*MaterialUniformValues
}

func (m *MaterialUniforms) Clear() {
	for _, values := range m.byMaterial {
		values.Clear()
	}
}

func (m *MaterialUniforms) Get(mat Material) *MaterialUniformValues {
	matType := reflect.TypeOf(mat)

	values, ok := m.byMaterial[matType]
	if ok {
		return values
	}

	ensureMapIsInitialized(&m.byMaterial)

	values = &MaterialUniformValues{
		Indices: map[byke.EntityId]uint32{},
	}

	m.byMaterial[matType] = values

	return values
}

func (m *MaterialUniforms) Upload(ctx *RenderContext) {
	for matType, values := range m.byMaterial {
		if len(values.Indices) == 0 {
			continue
		}

		// upload buffer to gpu
		label := matType.Name()
		values.Writer.WriteTo(ctx, &values.Buffer, label, wgpu.BufferUsageStorage)
	}
}

type MaterialUniformValues struct {
	Writer  wgsl.ArrayWriter
	Indices map[byke.EntityId]uint32

	Buffer *wgpu.Buffer
}

func (v *MaterialUniformValues) Clear() {
	v.Writer.Clear()
	clear(v.Indices)
}

func prepareMaterialUniforms(
	ctx *RenderContext,
	meshes ExtractedMeshes,
	uniforms *MaterialUniforms,
) {
	uniforms.Clear()

	for idx := range meshes.Meshes {
		item := &meshes.Meshes[idx]

		values := uniforms.Get(item.Material)

		// write material & store index for lookup
		index := uint32(values.Writer.ItemCount)
		item.Material.WriteUniforms(values.Writer.Next())
		values.Indices[item.EntityId] = index
	}

	uniforms.Upload(ctx)
}

type MaterialBindGroupHandleCache struct {
	handles map[any]*MaterialBindGroupHandle
	prevId  MaterialBindGroupId
}

func (c *MaterialBindGroupHandleCache) ResolveBindGroup[B comparable, PB interface {
	*B
	MaterialBindGroup
}](handle **MaterialBindGroupHandle, value B) {
	if h := *handle; h != nil {
		if canonical, ok := h.BindGroup.(PB); ok && *(*B)(canonical) == value {
			// only non-bind-group fields changed — keep the existing handle
			return
		}
	}

	existing, ok := c.handles[value]
	if ok {
		*handle = existing
		return
	}

	c.prevId += 1

	pb := any(&value).(PB)

	*handle = &MaterialBindGroupHandle{
		Id:               c.prevId,
		BindGroup:        pb,
		PipelineKey:      pb.PipelineKey(),
		OrderIndependent: pb.IsOrderIndependent(),
	}

	ensureMapIsInitialized(&c.handles)

	c.handles[value] = *handle
}

type isMaterialComponent[M isMaterialComponent[M]] interface {
	spoke.IsSupportsChangeDetectionComponent[M]
	Material
}

func resolveMaterialBindGroupsSystem[M isMaterialComponent[M], PM interface {
	*M
	isMaterialComponent[M]
}](
	cache *MaterialBindGroupHandleCache,
	query byke.Query[struct {
		_        byke.Or[byke.Added[M], byke.Changed[M]]
		Material query.Ref[M]
	}],
) {
	for item := range query.Items() {
		var m PM = item.Material.Value
		any(m).(ResolvableMaterial).ResolveBindGroup(cache)
	}
}

type MaterialBindGroups struct {
	cache tickCache[MaterialBindGroupId, *wgpu.BindGroup]
}

func (m *MaterialBindGroups) MustLookup(mat Material) *wgpu.BindGroup {
	bindGroup, ok := m.cache.Get(mat.BindGroup().Id)
	if !ok {
		panic(fmt.Errorf("no BindGroup found for material type %T", mat))
	}

	return bindGroup
}

func tickMaterialBindGroupsSystems(
	bindGroups *MaterialBindGroups,
) {
	bindGroups.cache.Tick()
}

// This must be on a per-material basis, as we need to reference the per-material uniforms.
//
//	this function generic on the material.
func prepareMaterialBindGroupsSystem(
	ctx *RenderContext,
	meshes *ExtractedMeshes,
	bindGroups *MaterialBindGroups,
	uniforms *MaterialUniforms,
	pipelines *meshPipelineCache,
	viewsQuery byke.Query[struct {
		_        byke.With[ViewTarget]
		EntityId byke.EntityId
	}],
) {
	// TODO access layout for mesh without view. While the pipeline might depend on
	//  the output format, the bind group layout should not.
	viewId := viewsQuery.MustFirst().EntityId

	for idx := range meshes.Meshes {
		item := &meshes.Meshes[idx]

		// we need to create one bind group per unique material key.
		matBindGroup := item.Material.BindGroup()

		if _, ok := bindGroups.cache.Get(matBindGroup.Id); !ok {
			label := reflect.TypeOf(item.Material).Name()

			values := uniforms.Get(item.Material)

			var bindings []wgpu.BindGroupEntry
			bindings = append(bindings, BindingBuffer(values.Buffer))
			bindings = append(bindings, matBindGroup.BindGroup.Bindings()...)

			pipeline, ok := pipelines.Get(meshPipelineCacheKey{viewId, item.EntityId})
			if !ok {
				panic("mesh pipeline not found")
			}

			bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:   label,
				Layout:  pipeline.BindGroupLayout(2),
				Entries: Sequential(bindings...),
			})

			bindGroups.cache.Add(matBindGroup.Id, bindGroup)
		}
	}
}

func tickMaterialAreaSystem[M Material](
	area *Area[M],
) {
	area.Tick()
}

func frontFaceOf(f wgpu.FrontFace) wgpu.FrontFace {
	switch f {
	case wgpu.FrontFaceCW, wgpu.FrontFaceCCW:
		return f

	default:
		return wgpu.FrontFaceCCW
	}
}
