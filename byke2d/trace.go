package byke2d

import (
	"sync/atomic"

	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var submitCount atomic.Int64

type RenderMetrics struct {
	Submit       int32
	WriteBuffer  int32
	WriteTexture int32
}

func (m *RenderMetrics) reset() {
	*m = RenderMetrics{}
}

type RenderContext struct {
	Metrics         RenderMetrics
	MipmapGenerator *mipmapGenerator
	*wx.Context
}

func (t *RenderContext) Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex {
	t.Metrics.Submit += 1
	return t.Context.Submit(commandBuffers...)
}

func (t *RenderContext) TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error) {
	t.Metrics.WriteBuffer += 1
	return t.Context.TryWriteBuffer(buffer, offset, data)
}

func (t *RenderContext) TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error) {
	t.Metrics.WriteTexture += 1
	return t.Context.TryWriteTexture(destination, data, dataLayout, writeSize)
}

func (t *RenderContext) WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte) {
	t.Metrics.WriteBuffer += 1
	t.Context.WriteBuffer(buffer, bufferOffset, data)
}

func (t *RenderContext) WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) {
	t.Metrics.WriteTexture += 1
	t.Context.WriteTexture(destination, data, dataLayout, writeSize)
}

type RenderQueue interface {
	Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex
	TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error)
	TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error)
	WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte)
	WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D)
}
