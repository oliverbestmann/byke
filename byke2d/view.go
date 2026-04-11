package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[ViewTarget]()

type ViewTarget struct {
	byke.Component[ViewTarget]

	// Size of the view target in pixels
	Size glm.Vec2f

	// The format of the target texture view
	Format wgpu.TextureFormat

	// The target to render to, must support Format.
	Target *wgpu.TextureView

	// The sample count of this view target.
	SampleCount uint32
}
