package wgsl

import "github.com/oliverbestmann/webgpu/wgpu"

type ArrayWriter struct {
	ItemCount int

	writer StructWriter
}

func (a *ArrayWriter) Next() *StructWriter {
	a.ItemCount += 1
	a.writer.Sync()
	return &a.writer
}

func (a *ArrayWriter) WriteTo(ctx RenderContext, buf **wgpu.Buffer, label string, usage wgpu.BufferUsage) {
	a.writer.Sync()
	a.writer.WriteTo(ctx, buf, label, usage)
}

func (a *ArrayWriter) Clear() {
	a.ItemCount = 0
	a.writer.Clear()
}
