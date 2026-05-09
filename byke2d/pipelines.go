package byke2d

import (
	"github.com/hashicorp/golang-lru/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type PipelineConfig interface {
	comparable
	Specialize(ctx *RenderContext) *wgpu.RenderPipeline
}

type Pipelines[C PipelineConfig] struct {
	renderContext *RenderContext
	cache         *lru.Cache[C, Pipeline]
}

func newPipelineCache[C PipelineConfig](renderContext *RenderContext) Pipelines[C] {
	cache, _ := lru.NewWithEvict[C, Pipeline](32, releasePipelineOnEvict)

	return Pipelines[C]{
		renderContext: renderContext,
		cache:         cache,
	}
}

func (Pipelines[C]) FromWorld(world *byke.World) Pipelines[C] {
	renderContext := byke.RequireResourceOf[RenderContext](world)
	return newPipelineCache[C](renderContext)
}

func (p Pipelines[C]) Specialize(config C) Pipeline {
	cached, ok := p.cache.Get(config)
	if ok {
		return cached
	}

	bglCache, _ := lru.NewWithEvict[uint32, *wgpu.BindGroupLayout](32, releaseBindGroupLayoutOnEvict)

	pipeline := Pipeline{
		pipeline:    config.Specialize(p.renderContext),
		layoutCache: bglCache,
	}

	p.cache.Add(config, pipeline)

	return pipeline
}

type Pipeline struct {
	pipeline    *wgpu.RenderPipeline
	layoutCache *lru.Cache[uint32, *wgpu.BindGroupLayout]
}

// Get returns the actual WGPU pipeline.
// You must NOT release the returned wgpu.RenderPipeline.
func (pc *Pipeline) Get() *wgpu.RenderPipeline {
	if !pc.pipeline.IsValid() {
		panic("cached pipeline was released")
	}

	return pc.pipeline
}

// GetBindGroupLayout returns a cached bind group layout.
// You must NOT release the returned wgpu.BindGroupLayout.
func (pc *Pipeline) GetBindGroupLayout(idx uint32) *wgpu.BindGroupLayout {
	bindGroup, ok := pc.layoutCache.Get(idx)
	if ok {
		if !bindGroup.IsValid() {
			panic("cached bindGroup was released")
		}

		return bindGroup
	}

	bindGroup = pc.pipeline.GetBindGroupLayout(idx)
	pc.layoutCache.Add(idx, bindGroup)

	return bindGroup
}

func releasePipelineOnEvict[C any](_ C, pipe Pipeline) {
	pipe.layoutCache.Purge()
	pipe.pipeline.Release()
}

func releaseBindGroupLayoutOnEvict(_ uint32, ev *wgpu.BindGroupLayout) {
	ev.Release()
}
