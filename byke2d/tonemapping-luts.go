package byke2d

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"

	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

//go:embed luts/tony_mc_mapface.raw.gz
var lutTonyMcMapfaceGZ []byte

//go:embed luts/AgX-default_contrast.raw.gz
var lutAgXGZ []byte

//go:embed luts/Blender_-11_12.raw.gz
var lutBlenderFilmicGZ []byte

type LutTexture struct {
	TextureView *wgpu.TextureView
	Sampler     *wgpu.Sampler
}

type TonemappingLutTextures struct {
	lutTonyMcMapface *LutTexture
	lutAgX           *LutTexture
	lutBlenderFilmic *LutTexture
}

func (t *TonemappingLutTextures) TonyMcMapface(ctx *RenderContext) LutTexture {
	return t.get(ctx, &t.lutTonyMcMapface, "TonyMcMapface", lutTonyMcMapfaceGZ, wgpu.TextureFormatRGB9E5Ufloat, 48, 48, 48, 4)
}

func (t *TonemappingLutTextures) AgX(ctx *RenderContext) LutTexture {
	return t.get(ctx, &t.lutAgX, "AgX", lutAgXGZ, wgpu.TextureFormatRGBA16Float, 32, 32, 32, 8)
}

func (t *TonemappingLutTextures) BlenderFilmic(ctx *RenderContext) LutTexture {
	return t.get(ctx, &t.lutBlenderFilmic, "BlenderFilmic", lutBlenderFilmicGZ, wgpu.TextureFormatRGBA16Float, 64, 64, 64, 8)
}

func (t *TonemappingLutTextures) get(ctx *RenderContext, tex **LutTexture, label string, data []byte, format wgpu.TextureFormat, width, height, depth, bpp uint32) LutTexture {
	if *tex != nil {
		return **tex
	}

	defer puffin.NewScope("LookupTable." + label).End()

	size := wgpu.Extent3D{
		Width:              width,
		Height:             height,
		DepthOrArrayLayers: depth,
	}

	texture := ctx.CreateTexture(&wgpu.TextureDescriptor{
		Label:         label,
		Usage:         wgpu.TextureUsageCopyDst | wgpu.TextureUsageTextureBinding,
		Dimension:     wgpu.TextureDimension3D,
		Size:          size,
		Format:        format,
		MipLevelCount: 1,
		SampleCount:   1,
	})

	defer texture.Release()

	textureView := texture.CreateView(nil)

	copyInfo := &wgpu.TexelCopyTextureInfo{
		Texture: texture,
		Origin:  wgpu.Origin3D{},
		Aspect:  wgpu.TextureAspectAll,
	}

	layout := &wgpu.TexelCopyBufferLayout{
		Offset:       0,
		BytesPerRow:  width * bpp,
		RowsPerImage: height,
	}

	data = t.decompress(data)
	ctx.WriteTexture(copyInfo, data, layout, &size)

	sampler := wx.CachedSampler(ctx.Device, wgpu.SamplerDescriptor{
		Label:         label,
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		AddressModeW:  wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeLinear,
		LodMinClamp:   0,
		LodMaxClamp:   32,
		MaxAnisotropy: 1,
	})

	*tex = &LutTexture{
		TextureView: textureView,
		Sampler:     sampler,
	}

	return **tex
}

func (t *TonemappingLutTextures) decompress(data []byte) []byte {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		panic(fmt.Errorf("failed to open lookup table: %w", err))
	}

	data, err = io.ReadAll(r)
	if err != nil {
		panic(fmt.Errorf("failed to read lookup table: %w", err))
	}

	_ = r.Close()

	return data
}
