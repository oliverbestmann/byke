package wgsl

import (
	"fmt"
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type StructWriter struct {
	writer
}

func (s *StructWriter) Clear() {
	s.reset()
}

func (s *StructWriter) Bytes() []byte {
	s.alignTo(s.align)
	return s.buf
}

func (s *StructWriter) AppendFloat32(value float32) {
	appendTo(&s.writer, value, 4, 4)
}

func (s *StructWriter) AppendInt(value int32) {
	appendTo(&s.writer, value, 4, 4)
}

func (s *StructWriter) AppendUint(value uint32) {
	appendTo(&s.writer, value, 4, 4)
}

func (s *StructWriter) AppendVec2f(value glm.Vec2f) {
	appendTo(&s.writer, value, 8, 8)
}

func (s *StructWriter) AppendVec3f(value glm.Vec3f) {
	appendTo(&s.writer, value, 16, 12)
}

func (s *StructWriter) AppendVec4f(value glm.Vec4f) {
	appendTo(&s.writer, value, 16, 16)
}

func (s *StructWriter) AppendMat2f(value glm.Mat4f) {
	appendTo(&s.writer, value.Components(), 8, 16)
}

func (s *StructWriter) AppendMat3f(value glm.Mat3f) {
	values := value.Components()
	appendTo(&s.writer, values[0], 16, 16)
	appendTo(&s.writer, values[1], 16, 16)
	appendTo(&s.writer, values[2], 16, 16)
}

func (s *StructWriter) AppendMat4f(value glm.Mat4f) {
	appendTo(&s.writer, value.Components(), 16, 4*16)
}

type InstanceWriter struct {
	writer

	expectedSize int
	count        int
}

func (s *InstanceWriter) StartNew(size int) {
	s.requireSize()
	s.expectedSize += size
	s.count += 1
}

func (s *InstanceWriter) Clear() {
	s.reset()
	s.count = 0
	s.expectedSize = 0
}

func (s *InstanceWriter) InstanceCount() int {
	s.requireSize()
	return s.count
}

func (s *InstanceWriter) ByteCount() uint64 {
	s.requireSize()
	return uint64(len(s.buf))
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
	appendTo(&s.writer, value, 1, 4)
}

func (s *InstanceWriter) AppendInt(value int32) {
	appendTo(&s.writer, value, 1, 4)
}

func (s *InstanceWriter) AppendUint(value uint32) {
	appendTo(&s.writer, value, 1, 4)
}

func (s *InstanceWriter) AppendVec2f(value glm.Vec2f) {
	appendTo(&s.writer, value, 1, 8)
}

func (s *InstanceWriter) AppendVec3f(value glm.Vec3f) {
	appendTo(&s.writer, value, 1, 12)
}

func (s *InstanceWriter) AppendVec4f(value glm.Vec4f) {
	appendTo(&s.writer, value, 1, 16)
}

type RenderContext interface {
	CreateBuffer(descriptor *wgpu.BufferDescriptor) *wgpu.Buffer
	WriteBuffer(buffer *wgpu.Buffer, offset uint64, data []byte)
}

func (s *InstanceWriter) WriteTo(ctx RenderContext, bufRef **wgpu.Buffer) {
	data := s.Bytes()

	buf := *bufRef

	if buf == nil || int(buf.GetSize()) < len(data) {
		buf.Release()

		bufferSize := max(256, len(data))

		buf = ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "Sprite Instances",
			Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageVertex,
			Size:  uint64(bufferSize),
		})

		*bufRef = buf
	}

	// upload data to buffer
	ctx.WriteBuffer(buf, 0, data)

}

type writer struct {
	buf   []byte
	align int
}

func (w *writer) reset() {
	w.buf = w.buf[:0]
}

func (w *writer) alignTo(align int) {
	for len(w.buf)%align != 0 {
		w.buf = append(w.buf, 0)
	}
}

func appendTo[T any](w *writer, value T, align, size int) {
	if unsafe.Sizeof(value) > uintptr(size) {
		panic("value is larger than its expected size")
	}

	w.alignTo(align)

	ptrValue := (*byte)(unsafe.Pointer(&value))
	bufValue := unsafe.Slice(ptrValue, unsafe.Sizeof(value))
	w.buf = append(w.buf, bufValue...)

	// add padding
	padding := int(unsafe.Sizeof(value)) - size
	if padding > 0 {
		w.buf = append(w.buf, make([]byte, padding)...)
	}

	w.align = max(w.align, align)
}
