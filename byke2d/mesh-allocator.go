package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

type MeshSlab struct {
	// the allocated buffer ids
	Index  *wgpu.Buffer
	Vertex *wgpu.Buffer

	// index of first item
	Start uint32

	// number of items
	Size uint32
}

type meshSlab struct {
	IndexId     uint32
	VertexId    uint32
	Start, Size uint32
}

type MeshAllocator struct {
	buffers   map[uint32]*wgpu.Buffer
	slabs     map[*Mesh]meshSlab
	allocator slabAllocator
}

func NewMeshAllocator() *MeshAllocator {
	return &MeshAllocator{
		buffers: map[uint32]*wgpu.Buffer{},
		slabs:   map[*Mesh]meshSlab{},
	}
}

func (m *MeshAllocator) Alloc(mesh *Mesh) {
	panic("not implemented")
}
