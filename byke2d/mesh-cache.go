package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type meshBuffers struct {
	// vertex buffer for this mesh
	Vertex *wgpu.Buffer

	// index buffer for this mesh
	Indices *wgpu.Buffer

	// more per vertex attributes
	Attributes []vertexAttributeBuffer

	// buffer that holds the morph attributes
	MorphAttributes *wgpu.Buffer
}

func (m *meshBuffers) Release() {
	m.Vertex.Release()
	m.Indices.Release()
	m.MorphAttributes.Release()

	for _, buf := range m.Attributes {
		buf.Buffer.Release()
	}
}

type vertexAttributeBuffer struct {
	Attribute VertexAttribute
	Buffer    *wgpu.Buffer
}

type meshCache struct {
	Context *RenderContext
	cache   tickCache[*Mesh, *meshBuffers]
}

//goland:noinspection GoMixedReceiverTypes
func meshCacheFromWorld(world *byke.World) meshCache {
	return meshCache{
		Context: byke.RequireResourceOf[RenderContext](world),
	}
}

func (m *meshCache) Upload(mesh *Mesh, forceUpload bool) bool {
	bufs, ok := m.cache.Get(mesh)
	if ok {
		if !forceUpload {
			return false
		}

		// TODO re-use memory if possible
		bufs.Release()
		bufs = &meshBuffers{}

	} else {
		bufs = &meshBuffers{}
	}

	bufs.Vertex = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh vertex buffer",
		Usage:    wgpu.BufferUsageVertex,
		Contents: wgpu.ToBytes(mesh.vertices),
	})

	bufs.Indices = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh index buffer",
		Usage:    wgpu.BufferUsageIndex,
		Contents: wgpu.ToBytes(mesh.indices),
	})

	for _, attr := range mesh.attributes {
		bufs.Attributes = append(bufs.Attributes, vertexAttributeBuffer{
			Attribute: attr.Attribute,
			Buffer: m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
				Label:    "mesh attr: " + attr.Attribute.Name,
				Usage:    wgpu.BufferUsageVertex,
				Contents: wgpu.ToBytes(attr.Value),
			}),
		})
	}

	if len(mesh.morphTargets) > 0 {
		attr := collectMorphAttributes(mesh.morphTargets)

		bufs.MorphAttributes = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "mesh morph attributes",
			Usage:    wgpu.BufferUsageStorage,
			Contents: attr,
		})
	}

	m.cache.Add(mesh, bufs)

	return true
}

func collectMorphAttributes(targets [][]MorphAttributes) []byte {
	var attrs wgsl.StructWriter

	for _, target := range targets {
		for _, attr := range target {
			attrs.AppendVec3f(attr.Position)
			attrs.AppendVec3f(attr.Normal)
			attrs.AppendVec3f(attr.Tangent)
		}
	}

	return attrs.Bytes()
}

func (m *meshCache) Reset() {
	m.cache.Tick()
}

func (m *meshCache) Get(mesh *Mesh) (*meshBuffers, bool) {
	return m.cache.Get(mesh)
}
