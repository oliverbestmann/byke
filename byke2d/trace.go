package byke2d

import (
	"fmt"
	"strings"
)

type RenderMetrics struct {
	CreateBindGroup       uint32
	CreateBindGroupLayout uint32
	CreateCommandEncoder  uint32
	CreateRenderPipeline  uint32
	CreateShaderModule    uint32
	Submit                uint32
	WriteBuffer           uint32
	WriteTexture          uint32

	// render encoder metrics
	SetPipeline      uint32
	SetVertexBuffer  uint32
	SetIndexBuffer   uint32
	SetBindGroup     uint32
	SetImmediates    uint32
	SetBlendConstant uint32
	Draw             uint32
	DrawIndexed      uint32
}

func (m *RenderMetrics) reset() {
	*m = RenderMetrics{}
}

func (m *RenderMetrics) String() string {
	var out strings.Builder

	writef := func(format string, args ...any) {
		_, _ = fmt.Fprintf(&out, format, args...)
		out.WriteByte('\n')
	}

	writef("RenderContext")
	writef("  CreateBindGroup:       %5d", m.CreateBindGroup)
	writef("  CreateBindGroupLayout: %5d", m.CreateBindGroupLayout)
	writef("  CreateCommandEncoder:  %5d", m.CreateCommandEncoder)
	writef("  CreateRenderPipeline:  %5d", m.CreateRenderPipeline)
	writef("  CreateShaderModule:    %5d", m.CreateShaderModule)
	writef("  Submit:                %5d", m.Submit)
	writef("  WriteBuffer:           %5d", m.WriteBuffer)
	writef("  WriteTexture:          %5d", m.WriteTexture)

	writef("")
	writef("RenderEncoder")
	writef("  Pipeline:      %5d", m.SetPipeline)
	writef("  IndexBuffer:   %5d", m.SetIndexBuffer)
	writef("  VertexBuffer:  %5d", m.SetVertexBuffer)
	writef("  BindGroup:     %5d", m.SetBindGroup)
	writef("  Immediates:    %5d", m.SetImmediates)
	writef("  BlendConstant: %5d", m.SetBlendConstant)
	writef("  Draw:          %5d", m.Draw)
	writef("  DrawIndexed:   %5d", m.DrawIndexed)

	return out.String()
}
