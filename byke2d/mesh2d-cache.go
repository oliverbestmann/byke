package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh2dBuffers struct {
	// vertex buffer for this mesh
	Vertex *wgpu.Buffer

	// index buffer for this mesh
	Indices *wgpu.Buffer

	// Other vertex attributes
	Attributes []vertexAttributeBuffer
}

func (m *mesh2dBuffers) Release() {
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

type mesh2dCache struct {
	Context *RenderContext
	cache   tickCache[*Mesh, *mesh2dBuffers]
}

//goland:noinspection GoMixedReceiverTypes
func mesh2dCacheFromWorld(world *byke.World) mesh2dCache {
	return mesh2dCache{
		Context: byke.RequireResourceOf[RenderContext](world),
	}
}

func (m *mesh2dCache) Upload(mesh *Mesh, forceUpload bool) *mesh2dBuffers {
	bufs, ok := m.cache.Get(mesh)
	if ok {
		if !forceUpload {
			return bufs
		}

		// TODO re-use memory if possible
		bufs.Release()
		bufs = &mesh2dBuffers{}

	} else {
		bufs = &mesh2dBuffers{}
	}

	bufs.Vertex = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh2d vertex buffer",
		Usage:    wgpu.BufferUsageVertex,
		Contents: wgpu.ToBytes(mesh.vertices),
	})

	bufs.Indices = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh2d index buffer",
		Usage:    wgpu.BufferUsageIndex,
		Contents: wgpu.ToBytes(mesh.indices),
	})

	for _, attr := range mesh.attributes {
		bufs.Attributes = append(bufs.Attributes, vertexAttributeBuffer{
			Attribute: attr.Attribute,
			Buffer: m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
				Label:    "mesh2d attr: " + attr.Attribute.Name,
				Usage:    wgpu.BufferUsageVertex,
				Contents: wgpu.ToBytes(attr.Value),
			}),
		})
	}

	m.cache.Add(mesh, bufs)

	return bufs
}

func (m *mesh2dCache) Reset() {
	m.cache.Tick()
}

func (m *mesh2dCache) Get(mesh *Mesh) (*mesh2dBuffers, bool) {
	return m.cache.Get(mesh)
}
