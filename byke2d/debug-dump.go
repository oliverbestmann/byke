package byke2d

import (
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func PluginDebugDumpCamera(app *byke.App) {
	// _ = os.Setenv("WGPU_FORCE_FALLBACK_ADAPTER", "1")

	app.AddSystems(PostRender, dumpCameraViewToTextureSystem)
}

func dumpCameraViewToTextureSystem(
	vt byke.VirtualTime,
	ctx *RenderContext,
	pipelines *PipelineCache,
	textureCache *TextureCache,
	viewsQuery byke.Query[struct {
		Target *ViewTarget
	}],
) {
	var calls []func()

	if vt.Frames%10 != 0 {
		return
	}

	log := slog.With(slog.Int("frame", vt.Frames))

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Blit"})
	defer enc.Release()

	for view := range viewsQuery.Items() {
		imageWidth := uint32(view.Target.Size[0])
		imageHeight := uint32(view.Target.Size[1])

		texture := textureCache.Allocate(&wgpu.TextureDescriptor{
			Label:     "DebugCameraTex",
			Usage:     wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageCopySrc,
			Dimension: wgpu.TextureDimension2D,
			Size: wgpu.Extent3D{
				Width:              imageWidth,
				Height:             imageHeight,
				DepthOrArrayLayers: 1,
			},
			Format:        wgpu.TextureFormatRGBA8UnormSrgb,
			MipLevelCount: 1,
			SampleCount:   1,
		})

		sourceView := view.Target.UnsampledTexture()

		pipeline := pipelines.Specialize(BlitConfig{Format: texture.Descriptor.Format})

		sampler := ctx.CreateSampler(wgpu.SamplerDescriptor{
			Label:        "Blit",
			AddressModeU: wgpu.AddressModeClampToEdge,
			AddressModeV: wgpu.AddressModeClampToEdge,
			AddressModeW: wgpu.AddressModeClampToEdge,
			MagFilter:    wgpu.FilterModeNearest,
			MinFilter:    wgpu.FilterModeNearest,
			MipmapFilter: wgpu.MipmapFilterModeNearest,
		})

		BlitTexture(ctx, enc, pipeline, sampler, sourceView, texture.TextureView)

		bufferSize := uint64(imageWidth * imageHeight * 4)

		buffer := ctx.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "DebugBuf",
			Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageMapRead,
			Size:  bufferSize,
		})

		enc.CopyTextureToBuffer(
			&wgpu.TexelCopyTextureInfo{
				Texture: texture.Texture,
				Origin:  wgpu.Origin3D{},
				Aspect:  wgpu.TextureAspectAll,
			},
			&wgpu.TexelCopyBufferInfo{
				Layout: wgpu.TexelCopyBufferLayout{
					BytesPerRow:  4 * imageWidth,
					RowsPerImage: imageHeight,
				},
				Buffer: buffer,
			},
			&wgpu.Extent3D{
				Width:              imageWidth,
				Height:             imageHeight,
				DepthOrArrayLayers: 1,
			},
		)

		calls = append(calls, func() {
			// copy to buffer
			log.Debug("Map buffer to download frame")

			buffer.MapAsync(wgpu.MapModeRead, 0, bufferSize, func(status wgpu.MapAsyncStatus) {

				defer buffer.Release()

				log.Debug("Mapped buffer to main memory", slog.String("status", status.String()))

				if status != wgpu.MapAsyncStatusSuccess {
					return
				}

				im := &image.NRGBA{
					Pix:    buffer.GetMappedRange(0, uint(bufferSize)),
					Stride: int(4 * imageWidth),
					Rect:   image.Rect(0, 0, int(imageWidth), int(imageHeight)),
				}

				fp, err := os.Create(fmt.Sprintf("/tmp/image-%05d.png", vt.Frames))
				if err != nil {
					slog.Warn("Failed to create output file", slog.String("error", err.Error()))
					return
				}

				err = png.Encode(fp, im)
				if err != nil {
					slog.Warn("Failed to write png file", slog.String("error", err.Error()))
					return
				}
			})
		})
	}

	// encode into a command buffer
	buf := enc.Finish(&wgpu.CommandBufferDescriptor{Label: "Blit"})
	defer buf.Release()

	ctx.Submit(buf)

	for _, call := range calls {
		call()
	}
}
