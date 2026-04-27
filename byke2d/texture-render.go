package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

type RenderTexture struct {
	*Texture
	RenderView *wgpu.TextureView
}

func AsRenderTexture(tex *Texture) *RenderTexture {
	renderView := tex.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           tex.Descriptor.Label + ".RenderView",
		Format:          tex.Descriptor.Format,
		Dimension:       wgpu.TextureViewDimension2D,
		BaseMipLevel:    0,
		MipLevelCount:   1,
		BaseArrayLayer:  0,
		ArrayLayerCount: 1,
		Aspect:          wgpu.TextureAspectAll,
	})

	return &RenderTexture{
		Texture:    tex,
		RenderView: renderView,
	}
}

func (rt *RenderTexture) Updated(ctx *RenderContext) {
	// re-generate mipmaps if the texture has any
	ctx.MipmapGenerator.Generate(rt.Texture)
}
