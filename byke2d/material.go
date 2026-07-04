package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type Material interface {
	Shader() *ShaderDef
	BindingsLayout() []wgpu.BindGroupLayoutEntry
	Bindings() []wgpu.BindGroupEntry
	WriteUniforms(w *wgsl.StructWriter)
	Key() CompareTo
}

func PluginMaterial2d[M Material](app *byke.App) {
	app.InsertResource(MaterialBindGroups[M]{})
	app.InsertResource(MaterialUniforms[M]{})

	app.AddSystems(Render, byke.System(extractMesh2dSystem[M]).InSet(RenderPhaseExtract))
}

func PluginMaterial3d[M Material](app *byke.App) {
	app.InsertResource(MaterialBindGroups[M]{})
	app.InsertResource(MaterialUniforms[M]{})

	app.AddSystems(Render, byke.
		System(extractMesh3dSystem[M]).
		InSet(RenderPhaseExtract))

	app.AddSystems(Render, byke.
		System(prepareMesh3dInstances[M]).
		InSet(RenderPhasePrepareResources))

	app.AddSystems(Render, byke.
		System(prepareMaterialBindGroupsSystem[M]).
		InSet(RenderPhasePrepareBindGroups))

}

type MaterialBindGroups[M Material] struct {
	tickCache[any, *wgpu.BindGroup]
}

func createMaterialBindGroup[M Material](
	ctx *RenderContext,
	material Material,
	uniforms *MaterialUniforms[M],
) *wgpu.BindGroup {
	label := fmt.Sprintf("material group: %T", material)

	var bindings []wgpu.BindGroupEntry
	bindings = append(bindings, BindingBuffer(uniforms.buffer))
	bindings = append(bindings, material.Bindings()...)

	var layout []wgpu.BindGroupLayoutEntry
	layout = append(layout, BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false))
	layout = append(layout, material.BindingsLayout()...)

	return ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   label,
		Layout:  ctx.CreateBindGroupLayout(SequentialLayoutWithLabel("Material", layout...)),
		Entries: Sequential(bindings...),
	})
}

type MaterialUniforms[M Material] struct {
	Writer wgsl.ArrayWriter
	buffer *wgpu.Buffer
}

func prepareMaterialBindGroupsSystem[M Material](
	ctx *RenderContext,
	meshes *ExtractedMesh3d,
	bindGroups *MaterialBindGroups[M],
	uniforms *MaterialUniforms[M],
) {
	var mZero M

	label := fmt.Sprintf("Material %T", mZero)
	uniforms.Writer.WriteTo(ctx, &uniforms.buffer, label, wgpu.BufferUsageStorage)
	uniforms.Writer.Clear()

	for _, mesh := range meshes.Meshes {
		// wrong material
		if _, ok := mesh.Material.(M); !ok {
			continue
		}

		key := mesh.Material.Key()

		if _, ok := bindGroups.Get(key); !ok {
			bindGroup := createMaterialBindGroup(ctx, mesh.Material, uniforms)
			bindGroups.Add(key, bindGroup)
		}
	}
}
