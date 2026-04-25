package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[ViewTarget]()

// TODO Some preparations for post processing and to support MSAA

type colorAttachment struct {
	HasContent    bool
	Texture       *wgpu.TextureView
	ResolveTarget *wgpu.TextureView
}

type ViewTarget struct {
	byke.Component[ViewTarget]

	// Size of the view target in pixels
	Size glm.Vec2f

	// The format of the target texture view
	Format wgpu.TextureFormat

	// The sample count for this view target.
	SampleCount uint32

	// Clear color to apply to the view target
	ClearColor wx.Color

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
			View:          target.Texture,
			ResolveTarget: target.ResolveTarget,
			LoadOp:        loadOp,
			StoreOp:       wgpu.StoreOpStore,
			ClearValue:    clearColor,
		}
	}

	return wgpu.RenderPassColorAttachment{
		View:          target.Texture,
		ResolveTarget: nil,
		LoadOp:        loadOp,
		StoreOp:       wgpu.StoreOpStore,
		ClearValue:    clearColor,
	}
}

func (m *ViewTarget) Switch() {
	m.active = (m.active + 1) % 2
}
