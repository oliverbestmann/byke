package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type mesh2dBuffers struct {
	// vertex buffer for this mesh
	Vertex *wgpu.Buffer

	// index buffer for this mesh
	Indices *wgpu.Buffer

	// Other vertex attributes
	Colors *wgpu.Buffer

	InUse bool
}

type mesh2dCache struct {
	Context *RenderContext
	meshes  map[uint32]*mesh2dBuffers
}

//goland:noinspection GoMixedReceiverTypes
func (mesh2dCache) FromWorld(world *byke.World) mesh2dCache {
	return mesh2dCache{
		Context: byke.RequireResourceOf[RenderContext](world),
		meshes:  map[uint32]*mesh2dBuffers{},
	}
}

func (m *mesh2dCache) Upload(id uint32, vertices []glm.Vec2f, indices []uint32, colors []Color) *mesh2dBuffers {
	if buf, ok := m.meshes[id]; ok {
		buf.InUse = true
		return buf
	}

	bufVertex := m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh2d vertex buffer",
		Usage:    wgpu.BufferUsageVertex,
		Contents: wgpu.ToBytes(vertices),
	})

	bufIndex := m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh2d index buffer",
		Usage:    wgpu.BufferUsageIndex,
		Contents: wgpu.ToBytes(indices),
	})

	buf := &mesh2dBuffers{
		Vertex:  bufVertex,
		Indices: bufIndex,
		InUse:   true,
	}

	if colors != nil {
		buf.Colors = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "mesh2d attr: colors",
			Usage:    wgpu.BufferUsageVertex,
			Contents: wgpu.ToBytes(colors),
		})
	}

	m.meshes[id] = buf

	return buf
}

func (m *mesh2dCache) Reset() {
	for id, buf := range m.meshes {
		if !buf.InUse {
			delete(m.meshes, id)

			buf.Indices.Release()
			buf.Vertex.Release()
		}

		buf.InUse = false
	}
}

func (m *mesh2dCache) Get(id uint32) (*mesh2dBuffers, bool) {
	buf, ok := m.meshes[id]
	if !ok {
		return nil, false
	}

	buf.InUse = true
	return buf, true
}
