package byke2d

import (
	"slices"
	"unsafe"

	"github.com/oliverbestmann/webgpu/wgpu"
)

type trackedBindGroup struct {
	Group          *wgpu.BindGroup
	DynamicOffsets []uint32
}

type trackedVertexBuffer struct {
	Buffer *wgpu.Buffer
	Offset uint64
	Size   uint64
}

type trackedImmediates struct {
	Offset uint32
	Data   trackedSlice
}

type trackedIndexBuffer struct {
	Buffer       *wgpu.Buffer
	BufferFormat wgpu.IndexFormat
	FormatOffset uint64
	FormatSize   uint64
}

type trackedSlice struct {
	Data unsafe.Pointer
	Size int
}

type TrackedRenderPassEncoder struct {
	*wgpu.RenderPassEncoder
	Metrics *RenderMetrics

	pipeline *wgpu.RenderPipeline

	immediates  trackedImmediates
	indexBuffer trackedIndexBuffer

	bindGroups    map[uint32]trackedBindGroup
	vertexBuffers map[uint32]trackedVertexBuffer

	blendColor *wgpu.Color
}

func (t *TrackedRenderPassEncoder) SetBindGroup(groupIndex uint32, group *wgpu.BindGroup, dynamicOffsets []uint32) {
	active := t.bindGroups[groupIndex]
	if active.Group == group && slices.Equal(active.DynamicOffsets, dynamicOffsets) {
		return
	}

	ensureMapIsInitialized(&t.bindGroups)

	t.bindGroups[groupIndex] = trackedBindGroup{
		Group:          group,
		DynamicOffsets: dynamicOffsets,
	}

	t.RenderPassEncoder.SetBindGroup(groupIndex, group, dynamicOffsets)
	t.Metrics.SetBindGroup += 1
}

func (t *TrackedRenderPassEncoder) SetBlendConstant(color *wgpu.Color) {
	if t.blendColor != nil && *t.blendColor == *color {
		return
	}

	t.blendColor = color
	t.RenderPassEncoder.SetBlendConstant(color)
	t.Metrics.SetBlendConstant += 1
}

func (t *TrackedRenderPassEncoder) SetImmediates(offset uint32, data []byte) {
	immediates := trackedImmediates{
		Offset: offset,
		Data:   toTrackedSlice(data),
	}

	if t.immediates == immediates {
		return
	}

	t.immediates = immediates
	t.RenderPassEncoder.SetImmediates(offset, data)
	t.Metrics.SetImmediates += 1
}

func (t *TrackedRenderPassEncoder) SetPipeline(pipeline *wgpu.RenderPipeline) {
	if t.pipeline == pipeline {
		return
	}

	t.pipeline = pipeline
	t.RenderPassEncoder.SetPipeline(pipeline)
	t.Metrics.SetPipeline += 1
}

func (t *TrackedRenderPassEncoder) SetIndexBuffer(buffer *wgpu.Buffer, format wgpu.IndexFormat, offset, size uint64) {
	indexBuffer := trackedIndexBuffer{
		Buffer:       buffer,
		BufferFormat: format,
		FormatOffset: offset,
		FormatSize:   size,
	}

	if t.indexBuffer == indexBuffer {
		return
	}

	t.indexBuffer = indexBuffer
	t.RenderPassEncoder.SetIndexBuffer(buffer, format, offset, size)
	t.Metrics.SetIndexBuffer += 1
}

func (t *TrackedRenderPassEncoder) SetVertexBuffer(slot uint32, buffer *wgpu.Buffer, offset, size uint64) {
	vertexBuffer := trackedVertexBuffer{
		Buffer: buffer,
		Offset: offset,
		Size:   size,
	}

	if t.vertexBuffers[slot] == vertexBuffer {
		return
	}

	ensureMapIsInitialized(&t.vertexBuffers)

	t.vertexBuffers[slot] = vertexBuffer
	t.RenderPassEncoder.SetVertexBuffer(slot, buffer, offset, size)
	t.Metrics.SetVertexBuffer += 1
}

func (t *TrackedRenderPassEncoder) DrawIndexed(indexCount, instanceCount, firstIndex uint32, baseVertex int32, firstInstance uint32) {
	t.RenderPassEncoder.DrawIndexed(indexCount, instanceCount, firstIndex, baseVertex, firstInstance)
	t.Metrics.DrawIndexed += 1
}

func ensureMapIsInitialized[K comparable, V any](m *map[K]V) {
	if *m == nil {
		*m = make(map[K]V)
	}
}

func toTrackedSlice[T any](offsets []T) trackedSlice {
	return trackedSlice{
		Data: unsafe.Pointer(unsafe.SliceData(offsets)),
		Size: len(offsets),
	}
}
