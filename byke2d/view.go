package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[ViewTarget]()

type ViewTarget struct {
	byke.Component[ViewTarget]

	Size glm.Vec2f

	// The format to render to the color attachments
	Format wgpu.TextureFormat

	// The target texture to render the final image to. This is normally backed by
	// the surface, or by the cameras texture, if the camera renderes to a texture.
	SurfaceTextureView *wgpu.TextureView

	// The format of the final surface
	SurfaceTextureFormat wgpu.TextureFormat

	// Samples to use for rendering to MainTextureView()
	SampleCount uint32

	// temporary textures to use for rendering, post processing, msaa write back, etc.
	attachments     [2]ViewTargetAttachment
	attachmentIndex uint8
}

type PostProcessing struct {
	Source *wgpu.TextureView
	Target *ViewTargetAttachment
}

func (m *ViewTarget) PostProcess() PostProcessing {
	sourceIdx := m.attachmentIndex
	m.attachmentIndex = (m.attachmentIndex + 1) % 2

	return PostProcessing{
		Source: m.attachments[sourceIdx].TextureView,
		Target: &m.attachments[m.attachmentIndex],
	}
}

// DiscardContent marks the content as discarded. The next call
// to Attachment() will use LoadOpClear to prevent loading
// of old data and just clear the buffer again.
func (m *ViewTarget) DiscardContent() {
	m.attachments[m.attachmentIndex].hasContent = false
}

// Attachment returns the currently active render attachment.
// This should be where we render to. If we have a multiple sample texture,
// this will not automatically resolve.
// You need to manually resolve to an UnsampledAttachment()
func (m *ViewTarget) Attachment() wgpu.RenderPassColorAttachment {
	return m.attachments[m.attachmentIndex].Attachment()
}

func (m *ViewTarget) SampledTexture() *wgpu.TextureView {
	return m.attachments[m.attachmentIndex].SampledView
}

func (m *ViewTarget) UnsampledTexture() *wgpu.TextureView {
	return m.attachments[m.attachmentIndex].TextureView
}

// UnsampledAttachment returns the currently active unsampled attachment.
func (m *ViewTarget) UnsampledAttachment() wgpu.RenderPassColorAttachment {
	return m.attachments[m.attachmentIndex].UnsampledAttachment()
}

type ViewTargetAttachment struct {
	TextureView *wgpu.TextureView

	// An optional multisample texture
	SampledView *wgpu.TextureView

	// The clear color of this attachment
	ClearColor Color

	hasContent bool
}

func (v *ViewTargetAttachment) Attachment() wgpu.RenderPassColorAttachment {
	if v.SampledView == nil {
		return v.UnsampledAttachment()
	}

	attachment := v.attachment(v.SampledView)
	attachment.ResolveTarget = v.TextureView
	return attachment
}

func (v *ViewTargetAttachment) UnsampledAttachment() wgpu.RenderPassColorAttachment {
	return v.attachment(v.TextureView)
}

func (v *ViewTargetAttachment) attachment(view *wgpu.TextureView) wgpu.RenderPassColorAttachment {
	var clearColor wgpu.Color
	var loadOp = wgpu.LoadOpLoad

	if !v.hasContent {
		v.hasContent = true

		loadOp = wgpu.LoadOpClear
		clearColor = colorToWGPU(v.ClearColor)
	}

	return wgpu.RenderPassColorAttachment{
		View:          view,
		ResolveTarget: nil,
		LoadOp:        loadOp,
		StoreOp:       wgpu.StoreOpStore,
		ClearValue:    clearColor,
	}
}

func colorToWGPU(c Color) wgpu.Color {
	r, g, b, a := c.Components()

	return wgpu.Color{
		R: float64(r),
		G: float64(g),
		B: float64(b),
		A: float64(a),
	}
}

func buildCameraViewTarget(textureCache *TextureCache, surfaceValues currentSurfaceValues, renderTarget RenderTarget, clearColor Color, hdr, msaa bool) (*ViewTarget, bool) {
	// hdr -> use float16 texture
	var format wgpu.TextureFormat

	if hdr {
		format = wgpu.TextureFormatRGBA16Float
	} else {
		format = wgpu.TextureFormatBGRA8Unorm
	}

	var width, height uint32
	var surfaceTextureView *wgpu.TextureView
	var surfaceTextureFormat wgpu.TextureFormat

	switch {
	case renderTarget.PrimaryWindow:
		width = surfaceValues.Texture.GetWidth()
		height = surfaceValues.Texture.GetHeight()
		surfaceTextureView = surfaceValues.TextureView
		surfaceTextureFormat = surfaceValues.Format

	case renderTarget.Texture != nil:
		width = renderTarget.Texture.Width()
		height = renderTarget.Texture.Height()
		surfaceTextureView = renderTarget.Texture.RenderView
		surfaceTextureFormat = renderTarget.Texture.Texture.Descriptor.Format

	default:
		// invalid configuration
		return nil, false
	}

	var sampleCount uint32 = 1

	a := textureCache.Allocate(&wgpu.TextureDescriptor{
		Label:     "CameraIntermediate",
		Usage:     wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
		Dimension: wgpu.TextureDimension2D,
		Format:    format,
		Size: wgpu.Extent3D{
			Width:              width,
			Height:             height,
			DepthOrArrayLayers: 1,
		},
		MipLevelCount: 1,
		SampleCount:   1,
	})

	b := textureCache.Allocate(&wgpu.TextureDescriptor{
		Label:     "CameraIntermediate",
		Usage:     wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
		Dimension: wgpu.TextureDimension2D,
		Format:    format,
		Size: wgpu.Extent3D{
			Width:              width,
			Height:             height,
			DepthOrArrayLayers: 1,
		},
		MipLevelCount: 1,
		SampleCount:   1,
	})

	var sampledTextureView *wgpu.TextureView

	if msaa {
		sampleCount = 4

		sampled := textureCache.Allocate(&wgpu.TextureDescriptor{
			Label:     "CameraIntermediate.Sampled",
			Usage:     wgpu.TextureUsageRenderAttachment,
			Dimension: wgpu.TextureDimension2D,
			Format:    format,
			Size: wgpu.Extent3D{
				Width:              width,
				Height:             height,
				DepthOrArrayLayers: 1,
			},
			MipLevelCount: 1,
			SampleCount:   4,
		})

		sampledTextureView = sampled.TextureView
	}

	view := &ViewTarget{
		Size:                 glm.Vec2f{float32(width), float32(height)},
		Format:               format,
		SurfaceTextureView:   surfaceTextureView,
		SurfaceTextureFormat: surfaceTextureFormat,
		SampleCount:          sampleCount,
		attachments: [2]ViewTargetAttachment{
			{
				TextureView: a.TextureView,
				SampledView: sampledTextureView,
				ClearColor:  clearColor,
			},
			{
				TextureView: b.TextureView,
				SampledView: sampledTextureView,
				ClearColor:  clearColor,
			},
		},
	}

	return view, true
}
