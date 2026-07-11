package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[MSAA]()

type MSAA struct {
	byke.ImmutableComponent[MSAA]
}

var multisampleStateOne = wgpu.MultisampleState{
	Count: 1,
	Mask:  0xffffffff,
}

func multisampleState(sampleCount uint32) wgpu.MultisampleState {
	return multisampleStateWithAlpha(sampleCount, false)
}

func multisampleStateWithAlpha(sampleCount uint32, alphaToCoverage bool) wgpu.MultisampleState {
	return wgpu.MultisampleState{
		Count:                  sampleCount,
		Mask:                   0xffffffff,
		AlphaToCoverageEnabled: alphaToCoverage,
	}
}
