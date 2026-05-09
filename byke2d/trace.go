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

func (rc *RenderContext) init(ctx *wx.Context) {
	rc.Context = ctx
	rc.MipmapGenerator = makeMipmapGenerator(rc)
}

func (rc *RenderContext) Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex {
	rc.Metrics.Submit += 1
	return rc.Context.Submit(commandBuffers...)
}

func (rc *RenderContext) TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error) {
	rc.Metrics.WriteBuffer += 1
	return rc.Context.TryWriteBuffer(buffer, offset, data)
}

func (rc *RenderContext) TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error) {
	rc.Metrics.WriteTexture += 1
	return rc.Context.TryWriteTexture(destination, data, dataLayout, writeSize)
}

func (rc *RenderContext) WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte) {
	rc.Metrics.WriteBuffer += 1
	rc.Context.WriteBuffer(buffer, bufferOffset, data)
}

func (rc *RenderContext) WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) {
	rc.Metrics.WriteTexture += 1
	rc.Context.WriteTexture(destination, data, dataLayout, writeSize)
}

type RenderQueue interface {
	Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex
	TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error)
	TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error)
	WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte)
	WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D)
}
