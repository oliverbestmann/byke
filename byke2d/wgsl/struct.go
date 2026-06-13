package wgsl

import (
	"unsafe"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type StructWriter struct {
	buf   []byte
	align int
}

func (s *StructWriter) Clear() {
	s.buf = s.buf[:0]
	s.align = 0
}

func (s *StructWriter) Bytes() []byte {
	s.AlignTo(s.align)
	return s.buf
}

func (s *StructWriter) AppendFloat32(value float32) {
	s.AlignTo(4)
	structAppend(s, value, 4)
}

func (s *StructWriter) AppendInt(value int32) {
	s.AlignTo(4)
	structAppend(s, value, 4)
}

func (s *StructWriter) AppendUint(value uint32) {
	s.AlignTo(4)
	structAppend(s, value, 4)
}

func (s *StructWriter) AppendVec2f(value glm.Vec2f) {
	s.AlignTo(8)
	structAppend(s, value, 8)
}

func (s *StructWriter) AppendVec3f(value glm.Vec3f) {
	s.AlignTo(16)
	structAppend(s, value, 12)
}

func (s *StructWriter) AppendVec4f(value glm.Vec4f) {
	s.AlignTo(16)
	structAppend(s, value, 16)
}

func (s *StructWriter) AppendMat2f(value glm.Mat4f) {
	s.AlignTo(8)
	structAppend(s, value.Column(0), 8)
	structAppend(s, value.Column(1), 8)
}

func (s *StructWriter) AppendMat3f(value glm.Mat3f) {
	s.AlignTo(16)
	structAppend(s, value.Column(0), 16)
	structAppend(s, value.Column(1), 16)
	structAppend(s, value.Column(2), 16)
}

func (s *StructWriter) AppendMat4f(value glm.Mat4f) {
	s.AlignTo(16)
	structAppend(s, value.Column(0), 16)
	structAppend(s, value.Column(1), 16)
	structAppend(s, value.Column(2), 16)
	structAppend(s, value.Column(3), 16)
}

func (s *StructWriter) WriteTo(ctx RenderContext, bufRef **wgpu.Buffer, usage wgpu.BufferUsage) {
	writeTo(ctx, bufRef, usage, s.Bytes())
}

func (s *StructWriter) AlignTo(align int) {
	if align == 0 {
		return
	}

	if pad := align - len(s.buf)%align; pad < align {
		zero := make([]byte, pad)
		s.buf = append(s.buf, zero...)
	}

	// struct align is the maximum alignment
	// of any individual field
	s.align = max(s.align, align)
}

func (s *StructWriter) Offset() uint32 {
	return uint32(len(s.buf))
}

func structAppend[T any](s *StructWriter, value T, size int) {
	if unsafe.Sizeof(value) > uintptr(size) {
		panic("value is larger than its expected size")
	}

	ptrValue := (*byte)(unsafe.Pointer(&value))
	bufValue := unsafe.Slice(ptrValue, unsafe.Sizeof(value))
	s.buf = append(s.buf, bufValue...)

	// add padding if necessary
	padding := int(unsafe.Sizeof(value)) - size
	if padding > 0 {
		s.buf = append(s.buf, make([]byte, padding)...)
	}
}
