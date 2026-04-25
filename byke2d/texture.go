package byke2d

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"

	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type Texture struct {
	Texture     *wgpu.Texture
	TextureView *wgpu.TextureView
	Descriptor  *wgpu.TextureDescriptor
	Sampler     *wgpu.Sampler
}

func (t *Texture) Size() glm.Vec2f {
	return glm.Vec2f{
		float32(t.Descriptor.Size.Width),
		float32(t.Descriptor.Size.Height),
	}
}

func (t *Texture) Width() uint32 {
	return t.Descriptor.Size.Width
}

func (t *Texture) Height() uint32 {
	return t.Descriptor.Size.Height
}

func (t *Texture) WritePixels(ctx *RenderContext, pixels []byte) {
	rect := wx.RectangleFromXYWH(0, 0, t.Width(), t.Height())

	t.WritePixelsToRect(ctx, WritePixelsOptions{
		Pixels: pixels,
		Region: rect,
	})
}

type WritePixelsOptions struct {
	Pixels   []byte
	Region   wx.Rectangle2u
	Stride   uint32
	MipLevel uint32
}

func (t *Texture) WritePixelsToRect(ctx *RenderContext, opts WritePixelsOptions) {
	region := wx.RectangleFromXYWH(0, 0, t.Width(), t.Height())

	// fail if not in rect
	if !region.Contains(opts.Region) {
		panic(fmt.Errorf("region %q not in texture %q", opts.Region, region))
	}

	if opts.Stride == 0 {
		// TODO use bpp of the texture
		opts.Stride = opts.Region.Width() * 4
	}

	layout := &wgpu.TexelCopyBufferLayout{
		Offset:       0,
		BytesPerRow:  opts.Stride,
		RowsPerImage: opts.Region.Height(),
	}

	size := &wgpu.Extent3D{
		Width:              opts.Region.Width(),
		Height:             opts.Region.Height(),
		DepthOrArrayLayers: 1,
	}

	dest := &wgpu.TexelCopyTextureInfo{
		Texture:  t.Texture,
		MipLevel: opts.MipLevel,
		Origin: wgpu.Origin3D{
			X: opts.Region.Min[0],
			Y: opts.Region.Min[1],
		},
		Aspect: wgpu.TextureAspectAll,
	}

	// send data to the gpu
	ctx.WriteTexture(dest, opts.Pixels, layout, size)

	// generate mip maps for this texture
	ctx.MipmapGenerator.Generate(t)
}

type NewTextureOptions struct {
	SamplerConfig
	Label  string
	Format wgpu.TextureFormat
	Width  uint32
	Height uint32
}

type SamplerConfig struct {
	AddressModeU wgpu.AddressMode
	AddressModeV wgpu.AddressMode
	FilterMode   wgpu.FilterMode
}

func (c *SamplerConfig) fillValues() {
	if c.AddressModeU == wgpu.AddressModeUndefined {
		c.AddressModeU = wgpu.AddressModeClampToEdge
	}

	if c.AddressModeV == wgpu.AddressModeUndefined {
		c.AddressModeV = wgpu.AddressModeClampToEdge
	}

	if c.FilterMode == wgpu.FilterModeUndefined {
		c.FilterMode = wgpu.FilterModeLinear
	}
}

func NewTexture(ctx *RenderContext, opts NewTextureOptions) *Texture {
	var sampleCount uint32 = 1

	desc := &wgpu.TextureDescriptor{
		Label:         opts.Label,
		Format:        opts.Format,
		SampleCount:   sampleCount,
		MipLevelCount: mipmapLevelCount(opts.Width, opts.Height),

		Dimension: wgpu.TextureDimension2D,
		Size: wgpu.Extent3D{
			Width:              opts.Width,
			Height:             opts.Height,
			DepthOrArrayLayers: 1,
		},

		// allow to do almost everything with this texture
		Usage: wgpu.TextureUsageTextureBinding |
			wgpu.TextureUsageRenderAttachment |
			wgpu.TextureUsageCopyDst |
			wgpu.TextureUsageCopySrc,
	}

	return NewTextureFromDesc(ctx, opts.SamplerConfig, desc)
}

// NewTextureFromDesc gives you full control and creates a texture directly from
// a texture descriptor
func NewTextureFromDesc(ctx *RenderContext, sampleConfig SamplerConfig, desc *wgpu.TextureDescriptor) *Texture {
	texture := ctx.CreateTexture(desc)

	// now create a default texture view
	textureView := texture.CreateView(nil)

	// fill missing config values
	sampleConfig.fillValues()

	// and the default sampler for this texture
	sampler := ctx.CreateSampler(&wgpu.SamplerDescriptor{
		Label:         desc.Label + ".Sampler",
		AddressModeU:  sampleConfig.AddressModeU,
		AddressModeV:  sampleConfig.AddressModeV,
		AddressModeW:  wgpu.AddressModeUndefined,
		MagFilter:     sampleConfig.FilterMode,
		MinFilter:     sampleConfig.FilterMode,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		LodMinClamp:   0,
		LodMaxClamp:   32,
		MaxAnisotropy: 1,
	})

	t := &Texture{
		Texture:     texture,
		TextureView: textureView,
		Descriptor:  desc,
		Sampler:     sampler,
	}

	return t
}

func DecodeTextureFromMemory(ctx *RenderContext, buf []byte, sampleConfig SamplerConfig, srgb bool) (*Texture, error) {
	src, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("decode image from memory: %w", err)
	}

	tex := NewTextureFromImage(ctx, src, sampleConfig, srgb)
	return tex, nil
}

// NewTextureFromImage creates a new Texture from the given golang image.Image instance.
func NewTextureFromImage(ctx *RenderContext, src image.Image, sampleConfig SamplerConfig, srgb bool) *Texture {
	iw, ih := src.Bounds().Dx(), src.Bounds().Dy()
	rgba := image.NewNRGBA(image.Rect(0, 0, iw, ih))

	draw.Draw(rgba, rgba.Bounds(), src, image.Point{}, draw.Src)

	format := wgpu.TextureFormatRGBA8Unorm
	if srgb {
		format = wgpu.TextureFormatRGBA8UnormSrgb
	}

	t := NewTexture(ctx, NewTextureOptions{
		SamplerConfig: sampleConfig,
		Format:        format,
		Width:         uint32(iw),
		Height:        uint32(ih),
		Label:         "TexFromImage",
	})

	t.WritePixels(ctx, rgba.Pix)

	return t
}
