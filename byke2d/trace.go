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
	writef("  CreateBindGroup:       % 4d", m.CreateBindGroup)
	writef("  CreateBindGroupLayout: % 4d", m.CreateBindGroupLayout)
	writef("  CreateCommandEncoder:  % 4d", m.CreateCommandEncoder)
	writef("  CreateRenderPipeline:  % 4d", m.CreateRenderPipeline)
	writef("  CreateShaderModule:    % 4d", m.CreateShaderModule)
	writef("  Submit:                % 4d", m.Submit)
	writef("  WriteBuffer:           % 4d", m.WriteBuffer)
	writef("  WriteTexture:          % 4d", m.WriteTexture)

	writef("")
	writef("RenderEncoder")
	writef("  Pipeline:      % 4d", m.SetPipeline)
	writef("  IndexBuffer:   % 4d", m.SetIndexBuffer)
	writef("  VertexBuffer:  % 4d", m.SetVertexBuffer)
	writef("  BindGroup:     % 4d", m.SetBindGroup)
	writef("  Immediates:    % 4d", m.SetImmediates)
	writef("  BlendConstant: % 4d", m.SetBlendConstant)
	writef("  DrawIndexed:   % 4d", m.DrawIndexed)

	return out.String()
}
