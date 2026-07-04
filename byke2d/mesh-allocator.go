package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/meh"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type MeshSlab struct {
	VertexLayout VertexLayout

	// the allocated buffer ids
	Indices  *wgpu.Buffer
	Vertices *wgpu.Buffer

	// optional, only if morph attributes data is defined
	MorphAttributes      *wgpu.Buffer
	MorphAttributesIndex uint32

	// index of first item
	FirstIndex  uint32
	FirstVertex uint32

	IndicesCount uint32
}

type MeshAllocator struct {
	context    *RenderContext
	slabs      map[*Mesh]meshSlab
	allocators meh.Map[VertexLayout, *bufferSlabAllocator]

	morphAttributesAlloc  *slabAllocator
	morphAttributesBuffer *wgpu.Buffer
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

	if slab.HasMorphAttributes {
		result.MorphAttributes = m.morphAttributesBuffer
		result.MorphAttributesIndex = slab.FirstMorphAttribute
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

		if existing.HasMorphAttributes {
			m.morphAttributesAlloc.Free(existing.MorphAttributesStart)
		}
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

	// if we have morph attributes, allocate those too
	if morphTargets := mesh.morphTargets; len(morphTargets) > 0 {
		m.ensureMorphAttributes()

		morph := collectMorphAttributes(morphTargets)

		morphStart, ok := m.morphAttributesAlloc.Alloc(uint32(len(morph)))
		if !ok {
			panic(fmt.Errorf("failed to allocate %d bytes for morph attributes", len(morph)))
		}

		// vec3f (with padding) is 16 byte. we have three of them
		const attributeSize = 3 * 4 * 4

		slab.HasMorphAttributes = true
		slab.MorphAttributesStart = morphStart
		slab.FirstMorphAttribute = morphStart / attributeSize

		// upload data
		m.context.WriteBuffer(m.morphAttributesBuffer, uint64(morphStart), morph)
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

func (m *MeshAllocator) ensureMorphAttributes() {
	if m.morphAttributesBuffer != nil {
		return
	}

	const bufferSize = 4 * 1024 * 1024

	bufMorphAttributes := m.context.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "MorphAttributes",
		Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageStorage,
		Size:  uint64(bufferSize),
	})

	m.morphAttributesAlloc = newSlabAllocator(bufferSize)
	m.morphAttributesBuffer = bufMorphAttributes
}

type meshSlab struct {
	VertexLayout VertexLayout

	// mesh version that is currently uploaded
	Version uint32

	Allocator *bufferSlabAllocator

	FirstIndex          uint32
	FirstVertex         uint32
	FirstMorphAttribute uint32
	IndicesCount        uint32

	// offsets into buffers
	VerticesStart uint32
	IndicesStart  uint32

	HasMorphAttributes   bool
	MorphAttributesStart uint32
}

type bufferSlabAllocator struct {
	AllocVertices *slabAllocator
	AllocIndices  *slabAllocator

	BufVertices *wgpu.Buffer
	BufIndices  *wgpu.Buffer
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
