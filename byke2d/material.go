package byke2d

import (
	"fmt"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
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
	// Shader returns the shader for the material in its current configuration
	Shader() *ShaderDef

	// BindingsLayout returns the layout of this material that is needed
	// to create the bind group.
	BindingsLayout() []wgpu.BindGroupLayoutEntry

	// Bindings return the Bindings for the BindingsLayout above.
	Bindings() []wgpu.BindGroupEntry

	// WriteUniforms write the non bind group related material data to a struct.
	// This might be the tint of a color material.
	WriteUniforms(w *wgsl.StructWriter)

	// BindGroupKey returns a key that is unique to this materials bind group.
	BindGroupKey() MaterialBindGroupKey

	// BindGroupLayoutKey returns key that uniquely identifies the BindGroupLayout
	// for this material. This will be used to deduplicate pipelines for this
	// material type.
	BindGroupLayoutKey() MaterialBindGroupLayoutKey

	// Specialize specializes the provided pipeline.
	Specialize(pipeline *RenderPipelineDescriptor)
}

type MaterialValues struct {
	// FrontFace defaults to wgpu.FrontFaceCCW
	FrontFace wgpu.FrontFace

	// AlphaMode decides on the way this material handles alpha values.
	AlphaMode AlphaMode

	// AlphaCutoff is used with AlphaModeMask to define the
	// cutoff for the alpha value.
	AlphaCutoff float32

	// DoubleSided enables double-sided lighting.
	// Need to flip the backface vertex in pixel shader
	DoubleSided bool
}

func (m MaterialValues) Specialize(pipeline *RenderPipelineDescriptor) {
	pipeline.Primitive.FrontFace = frontFaceOf(m.FrontFace)

	if m.DoubleSided {
		// disable culling so we can render both sides of the triangles
		pipeline.Primitive.CullMode = wgpu.CullModeNone
	}

	switch m.AlphaMode {
	case AlphaModeBlend:
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStateAlphaBlending
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse

	case Premultiplied:
		pipeline.Fragment.Targets[0].Blend = &wgpu.BlendStatePremultipliedAlphaBlending
		pipeline.DepthStencil.DepthWriteEnabled = wgpu.OptionalBoolFalse

	case AlphaModeAlphaToCoverage:
		pipeline.Multisample.AlphaToCoverageEnabled = true

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
	hash.Bool(m.DoubleSided)
	return hash
}

type MaterialBindGroupKey uint64

func (k MaterialBindGroupKey) SortValue() uint64 {
	return uint64(k)
}

type MaterialBindGroupLayoutKey uint64

func (k MaterialBindGroupLayoutKey) SortValue() uint64 {
	return uint64(k)
}

func pluginMaterialCommon(app *byke.App) {
	app.InsertResource(MaterialBindGroups{})
	app.InsertResource(MaterialUniforms{})

	app.AddSystems(PreRender, tickMaterialBindGroupsSystems)

	app.AddSystems(Render, byke.
		System(prepareMaterialUniforms).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMesh3dInstancesSystem).
		After(prepareMaterialUniforms).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMaterialBindGroupsSystem).
		InSet(RenderPhasePrepareBindGroups))
}

func PluginMaterial[M Material](app *byke.App) {
	app.AddSystems(Render, byke.
		System(extractMeshesSystem[M]).
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

type MaterialBindGroups struct {
	lookup map[MaterialBindGroupKey]*wgpu.BindGroup
}

func (m *MaterialBindGroups) MustLookup(mat Material) *wgpu.BindGroup {
	bindGroup := m.lookup[mat.BindGroupKey()]
	if bindGroup == nil {
		panic(fmt.Errorf("no BindGroup found for material type %T", mat))
	}

	return bindGroup
}

func tickMaterialBindGroupsSystems(
	bindGroups *MaterialBindGroups,
) {
	for _, bindGroup := range bindGroups.lookup {
		bindGroup.Release()
	}

	clear(bindGroups.lookup)
}

// This must be on a per-material basis, as we need to reference the per-material uniforms.
//
//	this function generic on the material.
func prepareMaterialBindGroupsSystem(
	ctx *RenderContext,
	meshes *ExtractedMeshes,
	bindGroups *MaterialBindGroups,
	uniforms *MaterialUniforms,
) {
	ensureMapIsInitialized(&bindGroups.lookup)

	for idx := range meshes.Meshes {
		item := &meshes.Meshes[idx]

		// we need to create one bind group per unique material key.
		key := item.Material.BindGroupKey()

		if _, ok := bindGroups.lookup[key]; !ok {
			label := reflect.TypeOf(item.Material).Name()

			values := uniforms.Get(item.Material)

			var bindings []wgpu.BindGroupEntry
			bindings = append(bindings, BindingBuffer(values.Buffer))
			bindings = append(bindings, item.Material.Bindings()...)

			var layout []wgpu.BindGroupLayoutEntry
			layout = append(layout, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false))
			layout = append(layout, item.Material.BindingsLayout()...)

			bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
				Label:   label,
				Layout:  ctx.CreateBindGroupLayout(SequentialLayoutWithLabel(label, layout...)),
				Entries: Sequential(bindings...),
			})

			bindGroups.lookup[key] = bindGroup
		}
	}
}

func frontFaceOf(f wgpu.FrontFace) wgpu.FrontFace {
	switch f {
	case wgpu.FrontFaceCW, wgpu.FrontFaceCCW:
		return f

	default:
		return wgpu.FrontFaceCCW
	}
}
