package byke2d

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/puffin-go"
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

// WritePixels is a convenience method to write 2d pixel data to a texture.
// The pixel data must match the the texture format.
func (t *Texture) WritePixels(ctx *RenderContext, pixels []byte) {
	rect := glm.RectuFromXYWH(0, 0, t.Width(), t.Height())

	t.WritePixelsToRect(ctx, WritePixelsOptions{
		Pixels: pixels,
		Region: rect,
	})
}

type WritePixelsOptions struct {
	Pixels   []byte
	Region   glm.Rectu
	Stride   uint32
	MipLevel uint32
	Layer    uint32
}

// WritePixelsToRect writes pixel data to a sub-region of a specific layer
// to your texture. You can even specify which mip level to write to.
//
// Setting a mip level prevents the function from re-generating mipmaps automatically.
func (t *Texture) WritePixelsToRect(ctx *RenderContext, opts ...WritePixelsOptions) {
	defer puffin.NewScope("texture.WritePixels").End()

	region := glm.RectuFromXYWH(0, 0, t.Width(), t.Height())

	var generateMipMaps = true

	for _, opt := range opts {
		// fail if not in rect
		if !region.Contains(opt.Region) {
			panic(fmt.Errorf("region %q not in texture %q", opt.Region, region))
		}

		if opt.Stride == 0 {
			bpp := t.Descriptor.Format.ByteSize()
			if bpp == 0 {
				panic(fmt.Errorf("unknown byte size for format %q", t.Descriptor.Format))
			}

			opt.Stride = opt.Region.Width() * bpp
		}

		layout := &wgpu.TexelCopyBufferLayout{
			Offset:       0,
			BytesPerRow:  opt.Stride,
			RowsPerImage: opt.Region.Height(),
		}

		size := &wgpu.Extent3D{
			Width:              opt.Region.Width(),
			Height:             opt.Region.Height(),
			DepthOrArrayLayers: 1,
		}

		dest := &wgpu.TexelCopyTextureInfo{
			Texture:  t.Texture,
			MipLevel: opt.MipLevel,
			Origin: wgpu.Origin3D{
				X: opt.Region.Min[0],
				Y: opt.Region.Min[1],
				Z: opt.Layer,
			},
			Aspect: wgpu.TextureAspectAll,
		}

		// send data to the gpu
		ctx.WriteTexture(dest, opt.Pixels, layout, size)

		if opt.MipLevel > 0 {
			generateMipMaps = false
		}
	}

	if generateMipMaps {
		// generate mip maps for this texture
		ctx.MipmapGenerator.Generate(t)
	}
}

// Share marks this texture as shared. Calling Release on a shared texture
// has no effect.
func (t *Texture) Share() *Texture {
	wgpu.Share(t.Sampler)
	wgpu.Share(t.TextureView)
	wgpu.Share(t.Texture)
	return t
}

// Release releases all resources associated with this texture if the
// texture is not marked as shared.
func (t *Texture) Release() {
	t.Sampler.Release()
	t.TextureView.Release()
	t.Texture.Release()
}

type NewTexture2dOptions struct {
	Label string

	// Format is the texture format to use
	Format wgpu.TextureFormat

	// Size of the texture in pixel
	Width  uint32
	Height uint32

	// Set the number of mipmap levels to generate. Use zero to
	// generate all levels.
	MipmapLevels uint32

	// Texture usage. The default is a lot:
	//   TextureBinding | RenderAttachment | CopyDst | CopySrc
	TextureUsage wgpu.TextureUsage

	// Config for the default sampler for this new texture
	SamplerConfig
}

// NewTexture2d is a convenience method to create a new 2d Texture instance.
// It uses appropriate default values.
func NewTexture2d(ctx *RenderContext, opts NewTexture2dOptions) *Texture {
	var sampleCount uint32 = 1

	mipmapLevels := mipmapLevelCount(opts.Width, opts.Height)
	if opts.MipmapLevels > 0 {
		mipmapLevels = opts.MipmapLevels
	}

	desc := &wgpu.TextureDescriptor{
		Label:         opts.Label,
		Format:        opts.Format,
		SampleCount:   sampleCount,
		MipLevelCount: mipmapLevels,

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

	return NewTextureFromDesc(ctx, NewTextureDescriptor{
		SamplerDescriptor:     opts.SamplerConfig.ToWGPU(desc.Label),
		TextureDescriptor:     desc,
		TextureViewDescriptor: nil,
	})
}

type NewTextureDescriptor struct {
	TextureDescriptor     *wgpu.TextureDescriptor
	TextureViewDescriptor *wgpu.TextureViewDescriptor
	SamplerDescriptor     *wgpu.SamplerDescriptor
}

// NewTextureFromDesc gives you full control and creates a texture directly from
// a TextureDescriptor, a TextureViewDescriptor and a SamplerConfig
func NewTextureFromDesc(ctx *RenderContext, desc NewTextureDescriptor) *Texture {
	texture := ctx.CreateTexture(desc.TextureDescriptor)

	// now create a default texture view
	textureView := texture.CreateView(desc.TextureViewDescriptor)

	// create the default sampler for the texture
	samplerDescriptor := desc.SamplerDescriptor
	if samplerDescriptor == nil {
		samplerDescriptor = SamplerConfig{}.ToWGPU(desc.TextureDescriptor.Label)
	}

	sampler := ctx.CreateSampler(*samplerDescriptor)

	return &Texture{
		Texture:     texture,
		TextureView: textureView,
		Descriptor:  desc.TextureDescriptor,
		Sampler:     sampler,
	}
}

type TextureFromImageOptions struct {
	Label string

	// Set to true to interpret the pixel data as srgb, not linear.
	SRGB bool

	// Usage defaults to RenderAttachment | TextureBinding | CopyDst.
	Usage wgpu.TextureUsage

	SamplerConfig
}

// NewTextureFromImage creates a new Texture from the given golang image.Image instance.
func NewTextureFromImage(ctx *RenderContext, src image.Image, opts TextureFromImageOptions) *Texture {
	defer puffin.NewScope("byke2d.NewTextureFromImage").End()

	return newTextureFromImagesRGBA(ctx, []image.Image{src}, TextureFromSourcesOptions{
		Label:            opts.Label,
		SRGB:             opts.SRGB,
		TextureDimension: wgpu.TextureDimension2D,
		ViewDimension:    wgpu.TextureViewDimension2D,
		Usage:            opts.Usage,
		SamplerConfig:    opts.SamplerConfig,
	})
}

type TextureFromImagesOptions struct {
	TextureFromImageOptions
	ViewDimension wgpu.TextureViewDimension
}

// NewTextureFromImages creates a new 2d layer texture from the given golang image.Image sequence
func NewTextureFromImages(ctx *RenderContext, sources []image.Image, opts TextureFromImagesOptions) *Texture {
	defer puffin.NewScope("byke2d.NewTextureFromImages").End()

	return newTextureFromImagesRGBA(ctx, sources, TextureFromSourcesOptions{
		Label:            opts.Label,
		SRGB:             opts.SRGB,
		TextureDimension: wgpu.TextureDimension2D,
		ViewDimension:    opts.ViewDimension,
		Usage:            opts.Usage,
		SamplerConfig:    opts.SamplerConfig,
	})
}

type TextureFromSourcesOptions struct {
	Label string

	// Set to true to interpret the pixel data as srgb, not linear.
	SRGB bool

	// TextureDimension and ViewDimension default to 2d. Must be set explicitly
	// for 2d array textures or cube maps.
	TextureDimension wgpu.TextureDimension
	ViewDimension    wgpu.TextureViewDimension

	// Usage defaults to RenderAttachment | CopyDst
	Usage wgpu.TextureUsage

	SamplerConfig
}

func newTextureFromImagesRGBA(ctx *RenderContext, sources []image.Image, opts TextureFromSourcesOptions) *Texture {
	if opts.TextureDimension == wgpu.TextureDimensionUndefined {
		opts.TextureDimension = wgpu.TextureDimension2D
	}

	if opts.ViewDimension == wgpu.TextureViewDimensionUndefined {
		opts.TextureDimension = wgpu.TextureDimension2D
	}

	if opts.Usage == wgpu.TextureUsageNone {
		opts.Usage = wgpu.TextureUsageRenderAttachment |
			wgpu.TextureUsageTextureBinding |
			wgpu.TextureUsageCopyDst
	}

	// get the target texture format
	format := wgpu.TextureFormatRGBA8Unorm
	if opts.SRGB {
		format = wgpu.TextureFormatRGBA8UnormSrgb
	}

	var width, height uint32

	var layers []*image.NRGBA
	var writes []WritePixelsOptions

	for idx, src := range sources {
		// convert pixel data to rgba
		iw, ih, rgba := toNRGBA(src)

		layers = append(layers, rgba)

		if idx == 0 {
			width = uint32(iw)
			height = uint32(ih)
		} else {
			if width != uint32(ih) || height != uint32(iw) {
				panic(fmt.Errorf(
					"not all layers have the same size, expected %dx%d, got %dx%d at layer %d",
					width, height, iw, ih, idx,
				))
			}
		}

		writes = append(writes, WritePixelsOptions{
			Pixels: rgba.Pix,
			Region: glm.RectuFromXYWH(0, 0, width, height),
			Layer:  uint32(idx),
		})
	}

	t := NewTextureFromDesc(ctx, NewTextureDescriptor{
		TextureDescriptor: &wgpu.TextureDescriptor{
			Label:     opts.Label,
			Format:    format,
			Usage:     opts.Usage,
			Dimension: opts.TextureDimension,
			Size: wgpu.Extent3D{
				Width:              width,
				Height:             height,
				DepthOrArrayLayers: uint32(len(sources)),
			},
			MipLevelCount: mipmapLevelCount(width, height),
			SampleCount:   1,
		},
		TextureViewDescriptor: &wgpu.TextureViewDescriptor{
			Label:           opts.Label,
			Format:          format,
			Dimension:       opts.ViewDimension,
			BaseMipLevel:    0,
			MipLevelCount:   mipmapLevelCount(width, height),
			BaseArrayLayer:  0,
			ArrayLayerCount: uint32(len(layers)),
			Aspect:          wgpu.TextureAspectAll,
		},

		SamplerDescriptor: opts.SamplerConfig.ToWGPU(opts.Label),
	})

	t.WritePixelsToRect(ctx, writes...)

	return t
}

func toNRGBA(src image.Image) (int, int, *image.NRGBA) {
	iw, ih := src.Bounds().Dx(), src.Bounds().Dy()

	if rgba, ok := src.(*image.NRGBA); ok {
		return iw, ih, rgba
	}

	rgba := image.NewNRGBA(image.Rect(0, 0, iw, ih))
	draw.Draw(rgba, rgba.Bounds(), src, image.Point{}, draw.Src)
	return iw, ih, rgba
}

type SamplerConfig struct {
	AddressModeU wgpu.AddressMode
	AddressModeV wgpu.AddressMode
	AddressModeW wgpu.AddressMode
	FilterMode   wgpu.FilterMode
	MipmapFilter wgpu.MipmapFilterMode
}

func (c SamplerConfig) ToWGPU(label string) *wgpu.SamplerDescriptor {
	if c.AddressModeU == wgpu.AddressModeUndefined {
		c.AddressModeU = wgpu.AddressModeClampToEdge
	}

	if c.AddressModeV == wgpu.AddressModeUndefined {
		c.AddressModeV = wgpu.AddressModeClampToEdge
	}

	if c.FilterMode == wgpu.FilterModeUndefined {
		c.FilterMode = wgpu.FilterModeLinear
	}

	if c.MipmapFilter == wgpu.MipmapFilterModeUndefined {
		c.MipmapFilter = wgpu.MipmapFilterModeLinear
	}

	return &wgpu.SamplerDescriptor{
		Label:        label,
		AddressModeU: c.AddressModeU,
		AddressModeV: c.AddressModeV,
		AddressModeW: c.AddressModeW,
		MagFilter:    c.FilterMode,
		MinFilter:    c.FilterMode,
		MipmapFilter: c.MipmapFilter,
	}
}
