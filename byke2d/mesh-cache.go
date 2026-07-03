package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type meshBuffers struct {
	// The vertex layout of data in the Vertex field
	VertexLayout VertexLayout

	// vertex buffer for this mesh
	Vertex *wgpu.Buffer

	// index buffer for this mesh
	Indices *wgpu.Buffer

	// buffer that holds the morph attributes
	MorphAttributes *wgpu.Buffer

	// the version that was uploaded
	Version uint32
}

func (m *meshBuffers) Release() {
	m.Vertex.Release()
	m.Indices.Release()
	m.MorphAttributes.Release()
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

func (m *meshCache) Upload(mesh *Mesh) bool {
	bufs, ok := m.cache.Get(mesh)
	if ok {
		if bufs.Version == mesh.version {
			// no upload needed, we're up to date
			return false
		}

		// TODO re-use memory if possible
		bufs.Release()
		bufs = &meshBuffers{}

	} else {
		bufs = &meshBuffers{}
	}

	vertices, layout := mesh.WriteVerticesTo(nil)

	bufs.Vertex = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "mesh vertex buffer",
		Usage:    wgpu.BufferUsageVertex,
		Contents: vertices,
	})

	bufs.VertexLayout = layout

	if mesh.indices != nil {
		bufs.Indices = m.Context.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "mesh index buffer",
			Usage:    wgpu.BufferUsageIndex,
			Contents: wgpu.ToBytes(mesh.indices),
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

	bufs.Version = mesh.version

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
