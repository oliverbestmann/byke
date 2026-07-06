package byke2d

import (
	"fmt"
	"log/slog"
	"slices"
	"sort"
)

// slabAllocator manages allocation and deallocation of variable-sized memory blocks
// from a fixed-size buffer. It maintains sorted lists of used and free memory chunks,
// automatically coalescing adjacent free chunks to reduce fragmentation.
type slabAllocator struct {
	totalSize uint32

	// used tracks all allocated memory regions
	used []memChunk

	// free tracks available memory regions, kept sorted and merged
	free []memChunk
}

func newSlabAllocator(size uint32) *slabAllocator {
	return &slabAllocator{
		totalSize: size,

		free: []memChunk{
			{
				StartAt: 0,
				Size:    size,
			},
		},
	}
}

func (m *slabAllocator) Alloc(size uint32) (startAt uint32, ok bool) {
	defer m.checkInvariants()

	for idx := range m.free {
		chunk := m.free[idx]

		if chunk.Size >= size {
			allocStartAt := chunk.StartAt

			// remove free space from the end of chunk
			if m.free[idx].Size == size {
				// delete the free chunk if it is now empty
				m.free = slices.Delete(m.free, idx, idx+1)
			} else {
				// remove allocated space from chunk
				m.free[idx].StartAt += size
				m.free[idx].Size -= size
			}

			// and create a new chunk for the memory we've allocated
			idx := sort.Search(len(m.used), func(i int) bool {
				return m.used[i].StartAt >= allocStartAt
			})

			// insert so that the slice stays sorted
			m.used = slices.Insert(m.used, idx, memChunk{
				StartAt: allocStartAt,
				Size:    size,
			})

			return allocStartAt, true
		}
	}

	return 0, false
}

func (m *slabAllocator) Free(allocStart uint32) {
	defer m.checkInvariants()

	// find the index of the allocation
	idx := sort.Search(len(m.used), func(i int) bool {
		return m.used[i].StartAt >= allocStart
	})

	if idx >= len(m.used) || m.used[idx].StartAt != allocStart {
		slog.Warn("Invalid address for free", slog.Uint64("startAt", uint64(allocStart)))
		return
	}

	allocSize := m.used[idx].Size

	// delete the entry
	m.used = slices.Delete(m.used, idx, idx+1)

	m.returnSpace(allocStart, allocSize)
}

func (m *slabAllocator) Grow(size uint32) (prevSize, newSize uint32) {
	prevSize = m.totalSize
	newSize = nextPowerOfTwo(m.totalSize + size)

	// extend by the new free space
	m.returnSpace(prevSize, newSize-prevSize)
	m.totalSize = newSize

	return prevSize, newSize
}

func (m *slabAllocator) returnSpace(allocStart, allocSize uint32) {
	// just append free item
	m.free = append(m.free, memChunk{
		StartAt: allocStart,
		Size:    allocSize,
	})

	// and sort the slice
	sort.Slice(m.free, func(i, j int) bool {
		return m.free[i].StartAt < m.free[j].StartAt
	})

	// now merge free items
	for idx := len(m.free) - 1; idx > 0; idx-- {
		if m.free[idx-1].NextStart() == m.free[idx].StartAt {
			// merge
			m.free[idx-1].Size += m.free[idx].Size

			// and remove
			m.free = slices.Delete(m.free, idx, idx+1)
		}
	}
}

func (m *slabAllocator) checkInvariants() {
	for idx := 1; idx < len(m.free); idx++ {
		if m.free[idx-1].NextStart() > m.free[idx].StartAt {
			panic(fmt.Errorf("non sorted free-list item found"))
		}

		if m.free[idx-1].NextStart() == m.free[idx].StartAt {
			panic(fmt.Errorf("non merged free-list item found"))
		}
	}

	for idx := 1; idx < len(m.used); idx++ {
		if m.used[idx-1].NextStart() > m.used[idx].StartAt {
			panic(fmt.Errorf("non sorted allocation found"))
		}
	}
}

// memChunk represents a contiguous region of memory with a starting offset and size.
type memChunk struct {
	// StartAt is the byte offset where this chunk begins in the buffer.
	StartAt uint32

	// Size is the byte size of this chunk.
	Size uint32
}

// NextStart returns the byte offset of the first byte after this chunk.
func (f *memChunk) NextStart() uint32 {
	return f.StartAt + f.Size
}
