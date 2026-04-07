package byke2d

import "github.com/oliverbestmann/webgpu/wgpu"

type Texture struct {
	Texture     *wgpu.Texture
	TextureView *wgpu.TextureView
	Sampler     *wgpu.Sampler
	Descriptor  *wgpu.TextureDescriptor
}
