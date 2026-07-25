package byke2d

import (
	"image"
	"log/slog"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type DebugFrameCallback func(frameIndex int, image *image.NRGBA) error

func PluginDebugDumpCamera(callback DebugFrameCallback) byke.Plugin {
	return func(app *byke.App) {
		app.InsertResource(callback)
		app.AddSystems(PostRender, dumpCameraViewToTextureSystem)
	}
}

func dumpCameraViewToTextureSystem(
	vt byke.VirtualTime,
	ctx *RenderContext,
	pipelines *PipelineCache,
	textureCache *TextureCache,
	frameCallback DebugFrameCallback,
	viewsQuery byke.Query[struct {
		Target *ViewTarget
	}],
) {
	var calls []func()

	frame := vt.Frames - 1
	if frame%10 != 0 {
		return
	}

	log := slog.With(slog.Int("frame", frame))

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

				if err := frameCallback(frame, im); err != nil {
					slog.Warn("Failed to handle frame dump", slog.String("error", err.Error()))
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
