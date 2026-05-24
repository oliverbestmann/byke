package byke2d

import (
	"log/slog"
	"slices"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type TextureCache struct {
	Context *RenderContext
	used    []*Texture
	unused  []*Texture
}

func TextureCacheFromWorld(world *byke.World) TextureCache {
	renderContext := byke.RequireResourceOf[RenderContext](world)
	return TextureCache{Context: renderContext}
}

func (t *TextureCache) Allocate(desc *wgpu.TextureDescriptor) *Texture {
	for idx, tex := range t.unused {
		if *tex.Descriptor == *desc {
			t.unused = slices.Delete(t.unused, idx, idx+1)
			t.used = append(t.used, tex)
			return tex
		}
	}

	// allocate a new texture
	tex := NewTextureFromDesc(t.Context, SamplerConfig{}, desc)
	t.used = append(t.used, tex)

	slog.Info("Allocated texture",
		slog.String("label", tex.Descriptor.Label),
		slog.Any("size", tex.Size()),
	)

	return tex
}

func (t *TextureCache) Reset() {
	for _, tex := range t.unused {
		slog.Info("Freeing texture",
			slog.String("label", tex.Descriptor.Label),
			slog.Any("size", tex.Size()),
		)

		tex.TextureView.Release()
		tex.Texture.Release()
	}

	// clear references and empty slice
	clear(t.unused)
	t.unused = t.unused[:0]

	// copy used to unused
	t.unused = append(t.unused, t.used...)

	// clear references and empty used
	clear(t.used)
	t.used = t.used[:0]
}
