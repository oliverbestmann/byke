package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/meh"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// MeshSlab represents the GPU-allocated resources for a single mesh.
// It contains the index and vertex buffers along with offsets and counts needed for rendering.
type MeshSlab struct {
	// VertexLayout describes the structure of vertex data in the buffer.
	VertexLayout VertexLayout

	// the allocated buffer ids
	// Indices points to the GPU buffer containing index data (triangle definitions).
	Indices *wgpu.Buffer

	// Vertices points to the GPU buffer containing vertex data (positions, normals, uvs, etc.).
	Vertices *wgpu.Buffer

	// MorphAttributes points to the GPU buffer containing blend shape data for skeletal animation.
	// optional, only if morph attributes data is defined
	MorphAttributes *wgpu.Buffer

	// MorphAttributesIndex is the offset index into the morph attributes buffer for this mesh.
	MorphAttributesIndex uint32

	// FirstIndex is the byte offset of the first index in the Indices buffer.
	FirstIndex uint32

	// FirstVertex is the byte offset of the first vertex in the Vertices buffer.
	FirstVertex uint32

	// IndicesCount is the number of indices in this mesh (must be a multiple of 3 for triangle rendering).
	IndicesCount uint32
}

// MeshAllocator manages GPU buffer allocation for meshes.
// It uses slab allocators to pack multiple meshes into large GPU buffers efficiently,
// reducing allocation overhead and improving GPU memory coherence.
type MeshAllocator struct {
	context *RenderContext

	// slabs maps each mesh to its allocated GPU resources
	slabs map[*Mesh]meshSlab

	// allocators manages separate vertex and index buffer allocations for each vertex layout
	allocators meh.Map[VertexLayout, *BufferAllocator]

	// allocator for index buffers
	indices *BufferAllocator

	// morphAttributes manages allocation of blend shape data
	morphAttributes *BufferAllocator
}

// NewMeshAllocator creates a new mesh allocator for the given render context.
func NewMeshAllocator(ctx *RenderContext) *MeshAllocator {
	return &MeshAllocator{
		context: ctx,
		slabs:   map[*Mesh]meshSlab{},
	}
}

func meshAllocatorFromWorld(world *byke.World) MeshAllocator {
	ctx := world.RequireResourceOf[RenderContext]()
	return *NewMeshAllocator(ctx)
}

// Get retrieves the allocated GPU resources for the given mesh.
// Returns false if the mesh has not been allocated yet.
func (m *MeshAllocator) Get(mesh *Mesh) (MeshSlab, bool) {
	slab, ok := m.slabs[mesh]
	if !ok {
		return MeshSlab{}, false
	}

	result := MeshSlab{
		VertexLayout: slab.VertexLayout,
		Indices:      m.indices.Buffer,
		Vertices:     slab.Allocator.Buffer,
		FirstIndex:   slab.FirstIndex,
		FirstVertex:  slab.FirstVertex,
		IndicesCount: slab.IndicesCount,
	}

	if slab.HasMorphAttributes {
		result.MorphAttributes = m.morphAttributes.Buffer
		result.MorphAttributesIndex = slab.FirstMorphAttribute
	}

	return result, true
}

// Alloc allocates or reallocates GPU buffer space for the given mesh.
// If the mesh has been modified since the last allocation, it will be reallocated.
// Returns true if the mesh was newly allocated or reallocated, false if it was already current.
func (m *MeshAllocator) Alloc(mesh *Mesh) bool {
	m.ensureAllocators()

	existing, ok := m.slabs[mesh]
	if ok {
		if existing.Version == mesh.version {
			return false
		}

		// mesh has changed, we need to reallocate
		alloc := m.getAllocator(existing.VertexLayout)
		alloc.Free(existing.VerticesStart)

		m.indices.Free(existing.IndicesStart)

		if existing.HasMorphAttributes {
			m.morphAttributes.Free(existing.MorphAttributesStart)
		}
	}

	// get an allocator for the current vertex layout
	layout := mesh.VertexLayout()
	alloc := m.getAllocator(layout)

	// allocate space for the vertices
	verticesStart := alloc.Alloc(uint32(mesh.VertexCount()) * layout.Size())

	if verticesStart%layout.Size() != 0 {
		panic("vertex data not aligned")
	}

	// allocate space for the indices
	indicesStart := m.indices.Alloc(uint32(len(mesh.indices)) * 4)

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
		morph := collectMorphAttributes(morphTargets)

		morphStart := m.morphAttributes.Alloc(uint32(len(morph)))

		// vec3f (with padding) is 16 byte. we have three of them
		const attributeSize = 3 * 4 * 4

		slab.HasMorphAttributes = true
		slab.MorphAttributesStart = morphStart
		slab.FirstMorphAttribute = morphStart / attributeSize

		// upload data
		m.context.WriteBuffer(m.morphAttributes.Buffer, uint64(morphStart), morph)
	}

	// store the allocation
	m.slabs[mesh] = slab

	// upload vertex data
	vertices, _ := mesh.WriteVerticesTo(nil)
	m.context.WriteBuffer(alloc.Buffer, uint64(slab.VerticesStart), vertices)

	// upload index data
	indices := wgpu.ToBytes(mesh.indices)
	m.context.WriteBuffer(m.indices.Buffer, uint64(slab.IndicesStart), indices)

	return true
}

func (m *MeshAllocator) getAllocator(layout VertexLayout) *BufferAllocator {
	allocator, ok := m.allocators.Get(layout)
	if !ok {
		allocator = NewBufferAllocator(m.context, "VertexBuffer", wgpu.BufferUsageVertex, 512*1024)
		m.allocators.Insert(layout, allocator)
	}

	return allocator
}

func (m *MeshAllocator) ensureAllocators() {
	if m.morphAttributes == nil {
		m.morphAttributes = NewBufferAllocator(
			m.context,
			"MorphAttributes",
			wgpu.BufferUsageStorage,
			512*1024,
		)
	}

	if m.indices == nil {
		m.indices = NewBufferAllocator(
			m.context,
			"IndexBuffer",
			wgpu.BufferUsageIndex,
			512*1024,
		)
	}
}

type meshSlab struct {
	VertexLayout VertexLayout

	// mesh version that is currently uploaded
	Version uint32

	Allocator *BufferAllocator

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
