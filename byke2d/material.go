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
}

type ComparableMaterial interface {
	comparable
	Material
}

type MeshExtract[T byke.IsComponent[T]] interface {
	byke.IsComponent[T]
	GetMesh() *Mesh
}

func PluginMaterial2d[M ComparableMaterial](app *byke.App) {
	app.AddSystems(Render, byke.System(extractMesh2dSystem[M]).InSet(RenderPhaseExtract))
}

func PluginMaterial3d[M ComparableMaterial](app *byke.App) {
	app.AddSystems(Render, byke.System(extractMesh3dSystem[M]).InSet(RenderPhaseExtract))
}

type MaterialBindGroups struct {
	tickCache[Material, *wgpu.BindGroup]
}

func createMaterialBindGroup(ctx *RenderContext, material Material) *wgpu.BindGroup {
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
		Layout:  ctx.CreateBindGroupLayout(SequentialLayoutWithLabel("Material", layout...)),
		Entries: Sequential(bindings...),
	})
}
