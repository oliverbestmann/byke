package byke2d

import (
	"github.com/oliverbestmann/webgpu/wgpu"
)

func (t RenderContext) Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex {
	return t.Context.Submit(commandBuffers...)
}

func (t RenderContext) TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error) {
	return t.Context.TryWriteBuffer(buffer, offset, data)
}

func (t RenderContext) TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error) {
	return t.Context.TryWriteTexture(destination, data, dataLayout, writeSize)
}

func (t RenderContext) WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte) {
	t.Context.WriteBuffer(buffer, bufferOffset, data)
}

func (t RenderContext) WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) {
	t.Context.WriteTexture(destination, data, dataLayout, writeSize)
}

type RenderQueue interface {
	Submit(commandBuffers ...*wgpu.CommandBuffer) wgpu.SubmissionIndex
	TryWriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte) (err error)
	TryWriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D) (err error)
	WriteBuffer(buffer *wgpu.Buffer, bufferOffset uint64, data []byte)
	WriteTexture(destination *wgpu.TexelCopyTextureInfo, data []byte, dataLayout *wgpu.TexelCopyBufferLayout, writeSize *wgpu.Extent3D)
}
