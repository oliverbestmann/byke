package bykebiten

import "github.com/oliverbestmann/byke"

var renderLayerZero = RenderLayersOf(0)

type RenderLayers struct {
	byke.Component[RenderLayers]
	Layers uint32
}

func (r RenderLayers) Intersects(other RenderLayers) bool {
	return r.Layers&other.Layers != 0
}

func RenderLayersOf(layers ...int) RenderLayers {
	var r RenderLayers

	for _, layer := range layers {
		if layer < 0 || layer > 31 {
			panic("invalid render layer")
		}

		r.Layers = r.Layers | (1 << layer)
	}

	return r
}
