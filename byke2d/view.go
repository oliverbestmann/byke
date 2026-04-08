package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[ViewTarget]()

type ViewTarget struct {
	byke.Component[ViewTarget]

	Format wgpu.TextureFormat

	// The target to render to, must support Format.
	Target *wgpu.TextureView
}

type TextureCache struct {
}

type textureCacheItem struct {
	Texture  *Texture
	LastUsed uint64
}
