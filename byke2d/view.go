package byke2d

import (
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type ViewTarget struct {
	// Size of the view target in pixels
	Size glm.Vec2f

	// The format of the target texture view
	Format wgpu.TextureFormat

	// The sample count for this view target.
	SampleCount uint32

	// Clear color to apply to the view target
	ClearColor wx.Color

	// An optional callback that cleanups the ViewTarget after rendering.
	// This can be used to free temporary resources
	CleanupCallback func()

	// multiple attachments to support post processing
	attachments [2]*colorAttachment
	active      int
}

// DiscardContent marks the content as discarded. The next call
// to ColorAttachment() will use LoadOpClear to prevent loading
// of old data and just clear the buffer again.
func (m *ViewTarget) DiscardContent() {
	m.attachments[m.active%2].HasContent = false
}

func (m *ViewTarget) ColorAttachment() wgpu.RenderPassColorAttachment {
	target := m.attachments[m.active%2]

	var clearColor wgpu.Color

	loadOp := wgpu.LoadOpLoad
	if !target.HasContent {
		target.HasContent = true
		loadOp = wgpu.LoadOpClear

		r, g, b, a := m.ClearColor.Components()
		clearColor.R = float64(r)
		clearColor.G = float64(g)
		clearColor.B = float64(b)
		clearColor.A = float64(a)
	}

	if target.ResolveTarget != nil {
		return wgpu.RenderPassColorAttachment{
			View:          target.TextureView,
			ResolveTarget: target.ResolveTarget,
			LoadOp:        loadOp,
			StoreOp:       wgpu.StoreOpStore,
			ClearValue:    clearColor,
		}
	}

	return wgpu.RenderPassColorAttachment{
		View:          target.TextureView,
		ResolveTarget: nil,
		LoadOp:        loadOp,
		StoreOp:       wgpu.StoreOpStore,
		ClearValue:    clearColor,
	}
}

func (m *ViewTarget) Switch() {
	m.active = (m.active + 1) % 2
}

type colorAttachment struct {
	HasContent    bool
	TextureView   *wgpu.TextureView
	ResolveTarget *wgpu.TextureView
}

// TODO
//   * only last step can write to swapchain
//   * single msaa render target if enabled
//   * no need for msaa render target if disabled, can use one of the post processing textures
//   * two post processing textures for ping/pong rendering
//   * render active post processing texture to swapchain
//   .
//   render target (render_attachment, msaa?, hdr?)
//   do msaa writeback, post processing texture 1 (hdr?, no msaa)
//   other post processing steps (hdr?)
//   tone mapping to swapchain
//

func buildCameraViewTarget(textureCache *TextureCache, surfaceValues currentSurfaceValues, renderTarget RenderTarget, clearColor wx.Color, msaa bool) (*ViewTarget, bool) {
	switch {
	case renderTarget.PrimaryWindow && msaa:
		sv := surfaceValues

		// allocate a temporary texture to render to
		msaaTexture := textureCache.Allocate(&wgpu.TextureDescriptor{
			Label:     "Camera.MSAA",
			Usage:     wgpu.TextureUsageRenderAttachment,
			Dimension: sv.Texture.GetDimension(),
			Format:    sv.Texture.GetFormat(),
			Size: wgpu.Extent3D{
				Width:              sv.Texture.GetWidth(),
				Height:             sv.Texture.GetHeight(),
				DepthOrArrayLayers: 1,
			},
			MipLevelCount: 1,
			SampleCount:   4,
		})

		viewTarget := ViewTarget{
			ClearColor:  clearColor,
			Format:      sv.Texture.GetFormat(),
			Size:        sv.Size,
			SampleCount: 4,

			attachments: [2]*colorAttachment{
				{
					TextureView:   msaaTexture.TextureView,
					ResolveTarget: sv.TextureView,
				},
			},
		}

		return &viewTarget, true

	case renderTarget.PrimaryWindow:
		sv := surfaceValues

		viewTarget := ViewTarget{
			ClearColor:  clearColor,
			Format:      sv.Texture.GetFormat(),
			SampleCount: sv.Texture.GetSampleCount(),
			Size:        sv.Size,

			attachments: [2]*colorAttachment{
				{
					TextureView:   sv.TextureView,
					ResolveTarget: nil,
				},
			},
		}

		return &viewTarget, true

	case renderTarget.Texture != nil && msaa:
		target := renderTarget.Texture

		// allocate a temporary texture to render to
		msaaTexture := textureCache.Allocate(&wgpu.TextureDescriptor{
			Label:         renderTarget.Texture.Descriptor.Label + ".MSAA",
			Usage:         wgpu.TextureUsageRenderAttachment,
			Dimension:     target.Descriptor.Dimension,
			Size:          target.Descriptor.Size,
			Format:        target.Descriptor.Format,
			MipLevelCount: 1,
			SampleCount:   4,
		})

		viewTarget := ViewTarget{
			ClearColor:  clearColor,
			Format:      target.Descriptor.Format,
			Size:        target.Size(),
			SampleCount: 4,

			attachments: [2]*colorAttachment{
				{
					TextureView:   msaaTexture.TextureView,
					ResolveTarget: target.RenderView,
				},
			},
		}

		return &viewTarget, true

	case renderTarget.Texture != nil:
		target := renderTarget.Texture

		viewTarget := ViewTarget{
			ClearColor:  clearColor,
			Format:      target.Descriptor.Format,
			SampleCount: target.Descriptor.SampleCount,
			Size:        target.Size(),

			attachments: [2]*colorAttachment{
				{
					TextureView:   target.RenderView,
					ResolveTarget: nil,
				},
			},
		}

		return &viewTarget, true
	}

	return nil, false
}
