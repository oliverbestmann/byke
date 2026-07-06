package byke2d

import (
	"fmt"
	"strconv"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type TextureAtlasAllocator struct {
	SamplerConfig SamplerConfig
	TextureFormat wgpu.TextureFormat
	textures      []textureWithSlices
}

func (t *TextureAtlasAllocator) Allocate(ctx *RenderContext, width, height uint32) (*Texture, glm.Rectu) {
	// round height up to next multiple of 4 to reduce number of bins
	height4 := height
	if d := height4 % 4; d > 0 {
		height4 += 4 - d
	}

	// find a slice of the expected height
	tex, slice := t.findSlice(ctx, height4, width)

	// extract the target region
	region := glm.RectuFromXYWH(slice.NextX, slice.Y, width, height)

	// consume space in this new slice
	slice.NextX += width
	slice.AvailableWidth -= width

	return tex, region
}

func (t *TextureAtlasAllocator) findSlice(ctx *RenderContext, height, width uint32) (*Texture, *textureSlice) {
	// find a matching slice that still has space
	for _, tex := range t.textures {
		for idx := range tex.Slices {
			slice := &tex.Slices[idx]
			if slice.Height == height && slice.AvailableWidth >= width {
				return tex.Texture, slice
			}
		}
	}

	// find the first texture that still has room
	for idx := range t.textures {
		tex := &t.textures[idx]

		if tex.Available >= height {
			// start a new slice
			slice := textureSlice{
				Y:              tex.NextY,
				AvailableWidth: tex.Texture.Width(),
				Height:         height,
			}

			tex.Slices = append(tex.Slices, slice)

			// remove space width
			tex.Available -= height
			tex.NextY += height

			// return reference to the new slice
			refSlice := &tex.Slices[len(tex.Slices)-1]
			return tex.Texture, refSlice
		}
	}

	t.allocateNewTexture(ctx)

	if tex, slice := t.findSlice(ctx, height, width); tex != nil {
		return tex, slice
	}

	// still no space?
	panic(fmt.Errorf("failed to allocate slice for height %d, width %d", height, width))
}

func (t *TextureAtlasAllocator) allocateNewTexture(ctx *RenderContext) {
	// allocate a new texture and try again
	texture := NewTexture(ctx, NewTextureOptions{
		SamplerConfig: t.SamplerConfig,
		Label:         "TextureAtlas." + strconv.Itoa(len(t.textures)),
		Format:        t.TextureFormat,
		Width:         2048,
		Height:        2048,
	})

	t.textures = append(t.textures, textureWithSlices{
		Texture:   texture,
		Available: texture.Width(),
	})
}

type textureWithSlices struct {
	Texture   *Texture
	Slices    []textureSlice
	NextY     uint32
	Available uint32
}

type textureSlice struct {
	Y, Height             uint32
	NextX, AvailableWidth uint32
}
