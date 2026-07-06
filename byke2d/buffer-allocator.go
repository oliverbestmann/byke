package byke2d

import (
	"log/slog"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type BufferAllocator struct {
	// The current buffer. It can change after each call to Alloc,
	// so you might need to rebind.
	Buffer *wgpu.Buffer

	context   *RenderContext
	allocator *slabAllocator
	label     string
}

func NewBufferAllocator(ctx *RenderContext, label string, usage wgpu.BufferUsage, size uint32) *BufferAllocator {
	buffer := ctx.CreateBuffer(&wgpu.BufferDescriptor{
		Label: label,
		Usage: usage | wgpu.BufferUsageCopySrc | wgpu.BufferUsageCopyDst,
		Size:  uint64(size),
	})

	return &BufferAllocator{
		Buffer:    buffer,
		context:   ctx,
		allocator: newSlabAllocator(size),
		label:     label,
	}
}

func (m *BufferAllocator) Alloc(size uint32) (addr uint32) {
	addr, ok := m.allocator.Alloc(size)
	if !ok {
		m.grow(size)

		return m.Alloc(size)
	}

	return addr
}

func (m *BufferAllocator) Free(addr uint32) {
	m.allocator.Free(addr)
}

func (m *BufferAllocator) grow(size uint32) {
	prevSize, newSize := m.allocator.Grow(size)

	slog.Debug(
		"Reallocate buffer",
		slog.String("label", m.label),
		slog.Int("prevSize", int(prevSize)),
		slog.Int("newSize", int(newSize)),
	)

	bufOld := m.Buffer

	// we need a new larger buffer
	bufNew := m.context.CreateBuffer(&wgpu.BufferDescriptor{
		Label: m.label,
		Usage: m.Buffer.GetUsage(),
		Size:  uint64(newSize),
	})

	enc := m.context.CreateCommandEncoder(nil)
	defer enc.Release()

	enc.CopyBufferToBuffer(bufOld, 0, bufNew, 0, uint64(prevSize))

	buf := enc.Finish(nil)
	defer buf.Release()

	m.context.Submit(buf)

	// release the old buffer with the new one
	m.Buffer.Release()
	m.Buffer = bufNew
}
