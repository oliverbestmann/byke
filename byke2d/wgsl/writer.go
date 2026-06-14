package wgsl

import (
	"fmt"
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type RenderContext interface {
	CreateBuffer(descriptor *wgpu.BufferDescriptor) *wgpu.Buffer
	WriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte)
}

type InstanceWriter struct {
	buf []byte

	expectedSize int
	count        int
}

func (s *InstanceWriter) Clear() {
	s.buf = s.buf[:0]
	s.count = 0
	s.expectedSize = 0
}

func (s *InstanceWriter) StartNew(size int) {
	s.requireSize()
	s.expectedSize += size
	s.count += 1
}

func (s *InstanceWriter) InstanceCount() int {
	s.requireSize()
	return s.count
}

func (s *InstanceWriter) Bytes() []byte {
	s.requireSize()
	return s.buf
}

func (s *InstanceWriter) requireSize() {
	if len(s.buf) != s.expectedSize {
		panic(fmt.Sprintf("expected size %d, got %d", s.expectedSize, len(s.buf)))
	}
}

func (s *InstanceWriter) AppendFloat32(value float32) {
	s.buf = rawAppendTo(s.buf, value)
}

func (s *InstanceWriter) AppendInt(value int32) {
	s.buf = rawAppendTo(s.buf, value)
}

func (s *InstanceWriter) AppendUint(value uint32) {
	s.buf = rawAppendTo(s.buf, value)
}

func (s *InstanceWriter) AppendVec2f(value glm.Vec2f) {
	s.buf = rawAppendTo(s.buf, value)
}

func (s *InstanceWriter) AppendVec3f(value glm.Vec3f) {
	s.buf = rawAppendTo(s.buf, value)
}

func (s *InstanceWriter) AppendVec4f(value glm.Vec4f) {
	s.buf = rawAppendTo(s.buf, value)
}

func (s *InstanceWriter) WriteTo(ctx RenderContext, bufRef **wgpu.Buffer, label string) {
	writeTo(ctx, bufRef, label, wgpu.BufferUsageVertex, s.Bytes())
}

func writeTo(ctx RenderContext, bufRef **wgpu.Buffer, label string, usage wgpu.BufferUsage, data []byte) {
	buf := *bufRef

	if buf == nil || int(buf.GetSize()) < len(data) {
		buf.Release()

		bufferSize := max(256, len(data))

		buf = ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: label,
			Usage: wgpu.BufferUsageCopyDst | usage,
			Size:  uint64(bufferSize),
		})

		*bufRef = buf
	}

	// upload data to buffer
	ctx.WriteBuffer(buf, 0, data)
}

func rawAppendTo[T any](buf []byte, value T) []byte {
	ptrValue := (*byte)(unsafe.Pointer(&value))
	bufValue := unsafe.Slice(ptrValue, unsafe.Sizeof(value))
	buf = append(buf, bufValue...)
	return buf
}
