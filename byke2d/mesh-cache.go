package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type meshBuffers struct {
	// vertex buffer for this mesh
	Vertex *wgpu.Buffer

	// index buffer for this mesh
	Indices *wgpu.Buffer

	// Other vertex attributes
	Attributes []vertexAttributeBuffer
}

func (m *meshBuffers) Release() {
	m.Vertex.Release()
	m.Indices.Release()

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

func (m *meshCache) Upload(mesh *Mesh, forceUpload bool) *meshBuffers {
	bufs, ok := m.cache.Get(mesh)
	if ok {
		if !forceUpload {
			return bufs
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

	m.cache.Add(mesh, bufs)

	return bufs
}

func (m *meshCache) Reset() {
	m.cache.Tick()
}

func (m *meshCache) Get(mesh *Mesh) (*meshBuffers, bool) {
	return m.cache.Get(mesh)
}
