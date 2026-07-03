package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/meh"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type MeshSlab struct {
	VertexLayout VertexLayout

	// the allocated buffer ids
	Indices  *wgpu.Buffer
	Vertices *wgpu.Buffer

	// index of first item
	FirstIndex  uint32
	FirstVertex uint32

	IndicesCount uint32
}

type MeshAllocator struct {
	context    *RenderContext
	slabs      map[*Mesh]meshSlab
	allocators meh.Map[VertexLayout, *bufferSlabAllocator]
}

func NewMeshAllocator(ctx *RenderContext) *MeshAllocator {
	return &MeshAllocator{
		context: ctx,
		slabs:   map[*Mesh]meshSlab{},
	}
}

func meshAllocatorFromWorld(world *byke.World) MeshAllocator {
	ctx := byke.RequireResourceOf[RenderContext](world)
	return *NewMeshAllocator(ctx)
}

func (m *MeshAllocator) Get(mesh *Mesh) (MeshSlab, bool) {
	slab, ok := m.slabs[mesh]
	if !ok {
		return MeshSlab{}, false
	}

	result := MeshSlab{
		VertexLayout: slab.VertexLayout,
		Indices:      slab.Allocator.BufIndices,
		Vertices:     slab.Allocator.BufVertices,
		FirstIndex:   slab.FirstIndex,
		FirstVertex:  slab.FirstVertex,
		IndicesCount: slab.IndicesCount,
	}

	return result, true
}

func (m *MeshAllocator) Alloc(mesh *Mesh) bool {
	existing, ok := m.slabs[mesh]
	if ok {
		if existing.Version == mesh.version {
			return false
		}

		// mesh has changed, we need to reallocate
		alloc := m.getAllocator(existing.VertexLayout)
		alloc.AllocVertices.Free(existing.VerticesStart)
		alloc.AllocIndices.Free(existing.IndicesStart)
	}

	// get an allocator for the current vertex layout
	layout := mesh.VertexLayout()
	alloc := m.getAllocator(layout)

	// allocate space for the vertices
	verticesStart, ok := alloc.AllocVertices.Alloc(uint32(mesh.VertexCount()) * layout.Size())
	if !ok {
		panic(fmt.Errorf("failed to allocate %d vertices", mesh.VertexCount()))
	}

	if verticesStart%layout.Size() != 0 {
		panic("vertex data not aligned")
	}

	// allocate space for the indices
	indicesStart, ok := alloc.AllocIndices.Alloc(uint32(len(mesh.indices)) * 4)
	if !ok {
		panic(fmt.Errorf("failed to allocate %d indices", len(mesh.indices)))
	}

	slab := meshSlab{
		VertexLayout:  layout,
		Version:       mesh.version,
		Allocator:     alloc,
		FirstIndex:    indicesStart / 4,
		FirstVertex:   verticesStart / layout.Size(),
		IndicesCount:  uint32(len(mesh.indices)),
		VerticesStart: verticesStart,
		IndicesStart:  indicesStart,
	}

	// store the allocation
	m.slabs[mesh] = slab

	// upload vertex data
	vertices, _ := mesh.WriteVerticesTo(nil)
	m.context.WriteBuffer(alloc.BufVertices, uint64(slab.VerticesStart), vertices)

	// upload index data
	indices := wgpu.ToBytes(mesh.indices)
	m.context.WriteBuffer(alloc.BufIndices, uint64(slab.IndicesStart), indices)

	return true
}

func (m *MeshAllocator) getAllocator(layout VertexLayout) *bufferSlabAllocator {
	allocator, ok := m.allocators.Get(layout)
	if !ok {
		bufferSize := uint32(64 * 1024 * 1024)

		bufVertex := m.context.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "VertexBuffer",
			Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageVertex,
			Size:  uint64(bufferSize),
		})

		bufIndex := m.context.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "IndexBuffer",
			Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageIndex,
			Size:  uint64(bufferSize),
		})

		allocator = &bufferSlabAllocator{
			AllocVertices: newSlabAllocator(bufferSize),
			AllocIndices:  newSlabAllocator(bufferSize),

			BufVertices: bufVertex,
			BufIndices:  bufIndex,
		}

		m.allocators.Insert(layout, allocator)
	}

	return allocator
}

type meshSlab struct {
	VertexLayout VertexLayout

	// mesh version that is currently uploaded
	Version uint32

	Allocator *bufferSlabAllocator

	FirstIndex   uint32
	FirstVertex  uint32
	IndicesCount uint32

	// offsets into buffers
	VerticesStart uint32
	IndicesStart  uint32
}

type bufferSlabAllocator struct {
	AllocVertices *slabAllocator
	AllocIndices  *slabAllocator

	BufVertices *wgpu.Buffer
	BufIndices  *wgpu.Buffer
}
