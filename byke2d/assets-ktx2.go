package byke2d

import (
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/ktx2"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type Ktx2Loader struct{}

func (i Ktx2Loader) Type() reflect.Type {
	return reflect.TypeFor[*Texture]()
}

func (i Ktx2Loader) Load(ctx LoadContext, r io.ReadSeekCloser) (any, error) {
	defer func() { _ = r.Close() }()

	var settings LoadTextureSettings
	if ctx.Settings != nil {
		settings = *ctx.Settings.(*LoadTextureSettings)
	}

	renderContext, ok := ctx.World.ResourceOf[RenderContext]()
	if !ok {
		return nil, errors.New("no RenderContext in world")
	}

	label := path.Base(ctx.Path)

	return LoadKTXToTexture(renderContext, r, settings.LoadKTXTextureOptions, label)
}

type LoadKTXTextureOptions struct {
	OverrideTextureDimension     wgpu.TextureDimension
	OverrideTextureViewDimension wgpu.TextureViewDimension
}

func LoadKTXToTexture(renderContext *RenderContext, r io.ReadSeeker, settings LoadKTXTextureOptions, label string) (*Texture, error) {
	k, err := ktx2.Open(r)
	if err != nil {
		return nil, fmt.Errorf("load ktx2 header: %w", err)
	}

	dim, dimView, err := ktx2MapTextureType(k)
	if err != nil {
		return nil, fmt.Errorf("mapping dimension: %w", err)
	}

	if settings.OverrideTextureDimension != 0 {
		dim = settings.OverrideTextureDimension
	}

	if settings.OverrideTextureViewDimension != 0 {
		dimView = settings.OverrideTextureViewDimension
	}

	format, ok := ktx2VkFormatToWGSL[k.Header.VkFormat]
	if !ok {
		return nil, fmt.Errorf("unsupported texture format: %q", k.Header.VkFormat)
	}

	faces, err := k.Faces()
	if err != nil {
		return nil, fmt.Errorf("load faces: %w", err)
	}

	var desc = NewTextureDescriptor{
		DisableAutoMipmaps: true,

		TextureDescriptor: &wgpu.TextureDescriptor{
			Label:     label,
			Usage:     wgpu.TextureUsageCopyDst | wgpu.TextureUsageTextureBinding,
			Dimension: dim,
			Size: wgpu.Extent3D{
				Width:              k.Header.PixelWidth,
				Height:             max(1, k.Header.PixelHeight),
				DepthOrArrayLayers: max(1, k.Header.PixelDepth) * max(1, k.Header.LayerCount) * k.Header.FaceCount,
			},
			Format:        format,
			MipLevelCount: max(1, k.Header.LevelCount),
			SampleCount:   1,
			ViewFormats:   nil,
		},
		TextureViewDescriptor: &wgpu.TextureViewDescriptor{
			Label:           label,
			Format:          format,
			Dimension:       dimView,
			BaseMipLevel:    0,
			MipLevelCount:   max(1, k.Header.LevelCount),
			BaseArrayLayer:  0,
			ArrayLayerCount: max(1, k.Header.LayerCount) * k.Header.FaceCount,
			Aspect:          wgpu.TextureAspectAll,
		},
		SamplerDescriptor: &wgpu.SamplerDescriptor{
			Label:        label,
			AddressModeU: wgpu.AddressModeClampToEdge,
			AddressModeV: wgpu.AddressModeClampToEdge,
			AddressModeW: wgpu.AddressModeClampToEdge,
			MagFilter:    wgpu.FilterModeLinear,
			MinFilter:    wgpu.FilterModeLinear,
			MipmapFilter: wgpu.MipmapFilterModeLinear,
			LodMinClamp:  0,
			LodMaxClamp:  32,
		},
	}

	texture := NewTextureFromDesc(renderContext, desc)

	for _, face := range faces {
		texture.WritePixelsToRect(renderContext, WritePixelsOptions{
			Pixels:   face.Buffer,
			Region:   glm.RectuFromXYWH(0, 0, face.Width, face.Height),
			Stride:   face.RowStride,
			MipLevel: face.Level,
			Layer:    face.Layer3d,
		})
	}

	return texture, nil
}

func ktx2MapTextureType(k ktx2.Reader) (wgpu.TextureDimension, wgpu.TextureViewDimension, error) {
	tt, ok := k.TextureType()
	if !ok {
		return 0, 0, errors.New("unknown texture type")
	}

	switch tt {
	case ktx2.TextureType1d:
		return wgpu.TextureDimension1D, wgpu.TextureViewDimension1D, nil

	case ktx2.TextureType2d:
		return wgpu.TextureDimension2D, wgpu.TextureViewDimension2D, nil

	case ktx2.TextureType3d:
		return wgpu.TextureDimension3D, wgpu.TextureViewDimension3D, nil

	case ktx2.TextureTypeCubemap:
		return wgpu.TextureDimension2D, wgpu.TextureViewDimensionCube, nil

	case ktx2.TextureType1dArray:
		return wgpu.TextureDimension2D, wgpu.TextureViewDimension2D, nil

	case ktx2.TextureType2dArray:
		return wgpu.TextureDimension3D, wgpu.TextureViewDimension2DArray, nil

	case ktx2.TextureType3dArray:
		return wgpu.TextureDimension3D, wgpu.TextureViewDimension3D, nil

	case ktx2.TextureTypeCubemapArray:
		return wgpu.TextureDimension2D, wgpu.TextureViewDimensionCubeArray, nil

	default:
		return 0, 0, fmt.Errorf("unsupported texture format %q", tt)
	}
}

func (i Ktx2Loader) Extensions() []string {
	return []string{".ktx2"}
}

var ktx2VkFormatToWGSL = map[ktx2.VkFormat]wgpu.TextureFormat{
	ktx2.VK_FORMAT_E5B9G9R9_UFLOAT_PACK32: wgpu.TextureFormatRGB9E5Ufloat,
	ktx2.VK_FORMAT_B8G8R8A8_SRGB:          wgpu.TextureFormatBGRA8UnormSrgb,
	ktx2.VK_FORMAT_R8G8B8A8_SRGB:          wgpu.TextureFormatRGBA8UnormSrgb,
	ktx2.VK_FORMAT_R16G16B16A16_SFLOAT:    wgpu.TextureFormatRGBA16Float,
	ktx2.VK_FORMAT_R16G16_SFLOAT:          wgpu.TextureFormatRG16Float,
}
