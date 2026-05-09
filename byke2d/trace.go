package byke2d

type RenderMetrics struct {
	CreateBindGroup       int32
	CreateBindGroupLayout int32
	CreateCommandEncoder  int32
	CreateRenderPipeline  int32
	CreateShaderModule    int32
	Submit                int32
	WriteBuffer           int32
	WriteTexture          int32
}

func (m *RenderMetrics) reset() {
	*m = RenderMetrics{}
}
