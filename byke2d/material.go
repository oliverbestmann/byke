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
	// Multiple materials can have different uniform values with the same BindGroupKey.
	// The returned value must be comparable (i.e. can be used in a hashmap)
	BindGroupKey() MaterialBindGroupKey

	// Specialize specializes the provided pipeline.
	Specialize(pipeline *RenderPipelineDescriptor)
}

type MaterialBindGroupKey interface {
	SortValue() uint64
}

func pluginMaterialCommon(app *byke.App) {
	app.InsertResource(MaterialBindGroups{})

	app.AddSystems(PreRender, tickMaterialBindGroupsSystems)
}

func PluginMaterial2d[M Material](app *byke.App) {
	app.InsertResource(MaterialUniforms[M]{})

	app.AddSystems(Render, byke.System(extractMesh2dSystem[M]).InSet(RenderPhaseExtract))

	app.AddSystems(Render, byke.
		System(prepareMaterialUniforms[M]).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMaterialBindGroupsSystem[M]).
		InSet(RenderPhasePrepareBindGroups))
}

func PluginMaterial3d[M Material](app *byke.App) {
	app.InsertResource(MaterialBindGroups{})
	app.InsertResource(MaterialUniforms[M]{})

	app.AddSystems(Render, byke.
		System(extractMesh3dSystem[M]).
		InSet(RenderPhaseExtract))

	app.AddSystems(Render, byke.
		System(prepareMaterialUniforms[M], prepareMesh3dInstancesSystem[M]).
		Chain().
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMaterialBindGroupsSystem[M]).
		InSet(RenderPhasePrepareBindGroups))

}

type MaterialUniforms[M Material] struct {
	Writer  wgsl.ArrayWriter
	Indices map[byke.EntityId]uint32

	buffer *wgpu.Buffer
}

func prepareMaterialUniforms[M Material](
	ctx *RenderContext,
	meshes ExtractedMeshes3d,
	uniforms *MaterialUniforms[M],
) {
	ensureMapIsInitialized(&uniforms.Indices)

	uniforms.Writer.Clear()

	for idx := range meshes.Meshes {
		item := &meshes.Meshes[idx]

		if _, ok := item.Material.(M); !ok {
			// not the right material
			continue
		}

		// write material & store index for lookup
		index := uint32(uniforms.Writer.ItemCount)
		item.Material.WriteUniforms(uniforms.Writer.Next())
		uniforms.Indices[item.EntityId] = index
	}

	// upload buffer to gpu
	label := reflect.TypeFor[M]().Name()
	uniforms.Writer.WriteTo(ctx, &uniforms.buffer, label, wgpu.BufferUsageStorage)
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
// TODO evaluate if we would like to use a generic map[MaterialType]MaterialUniforms to not make
//
//	this function generic on the material.
func prepareMaterialBindGroupsSystem[M Material](
	ctx *RenderContext,
	meshes *ExtractedMeshes3d,
	bindGroups *MaterialBindGroups,
	uniforms *MaterialUniforms[M],
) {
	ensureMapIsInitialized(&bindGroups.lookup)

	for idx := range meshes.Meshes {
		item := &meshes.Meshes[idx]

		if _, ok := item.Material.(M); !ok {
			// not the right material
			continue
		}

		// we need to create one bind group per unique material key.
		key := item.Material.BindGroupKey()

		if _, ok := bindGroups.lookup[key]; !ok {
			label := reflect.TypeOf(item.Material).Name()

			var bindings []wgpu.BindGroupEntry
			bindings = append(bindings, BindingBuffer(uniforms.buffer))
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
