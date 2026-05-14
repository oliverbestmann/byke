package byke2d

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type ShaderValues = pre.Values

type PipelineContext interface {
	// Shader compiles a shader to a *wgpu.ShaderModule.
	Shader(label, shaderCode string, values ShaderValues) *wgpu.ShaderModule
}

type PipelineConfig interface {
	comparable
	Specialize(ctx PipelineContext) RenderPipelineDescriptor
}

type RenderPipelineDescriptor struct {
	Label        string
	Layout       []wgpu.BindGroupLayoutDescriptor
	Vertex       wgpu.VertexState
	Primitive    wgpu.PrimitiveState
	DepthStencil *wgpu.DepthStencilState
	Multisample  wgpu.MultisampleState
	Fragment     *wgpu.FragmentState
}

type Pipelines[C PipelineConfig] struct {
	renderContext *RenderContext
	pipelineCache *PipelineCache
	cache         *lru.Cache[C, Pipeline]
}

func newPipelines[C PipelineConfig](renderContext *RenderContext, pipelineCache *PipelineCache) Pipelines[C] {
	cache, _ := lru.NewWithEvict[C, Pipeline](32, releasePipelineOnEvict)

	return Pipelines[C]{
		renderContext: renderContext,
		cache:         cache,
		pipelineCache: pipelineCache,
	}
}

func (Pipelines[C]) FromWorld(world *byke.World) Pipelines[C] {
	renderContext := byke.RequireResourceOf[RenderContext](world)
	pipelineCache := byke.RequireResourceOf[PipelineCache](world)
	return newPipelines[C](renderContext, pipelineCache)
}

func (p Pipelines[C]) Specialize(config C) Pipeline {
	cached, ok := p.cache.Get(config)
	if ok {
		return cached
	}

	// create the pipeline descriptor
	desc := config.Specialize(p.pipelineCache)

	// create bind group layouts
	var bgls []*wgpu.BindGroupLayout
	for _, bgld := range desc.Layout {
		bgl := p.pipelineCache.BindGroupLayout(bgld)
		bgls = append(bgls, bgl)
	}

	// create the pipeline layout
	layout := p.renderContext.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            desc.Label,
		BindGroupLayouts: bgls,
	})

	defer layout.Release()

	// now create the actual pipeline
	pipe := p.renderContext.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:        desc.Label,
		Layout:       layout,
		Vertex:       desc.Vertex,
		Primitive:    desc.Primitive,
		DepthStencil: desc.DepthStencil,
		Multisample:  desc.Multisample,
		Fragment:     desc.Fragment,
	})

	pipeline := Pipeline{
		pipeline:         pipe,
		bindGroupLayouts: bgls,
	}

	p.cache.Add(config, pipeline)

	return pipeline
}

type Pipeline struct {
	pipeline         *wgpu.RenderPipeline
	bindGroupLayouts []*wgpu.BindGroupLayout
}

// Get returns the actual WGPU pipeline.
// You must NOT release the returned wgpu.RenderPipeline.
func (pc *Pipeline) Get() *wgpu.RenderPipeline {
	if !pc.pipeline.IsValid() {
		panic("cached pipeline was released")
	}

	return pc.pipeline
}

// BindGroupLayout returns a cached bind group layout.
// You must NOT release the returned wgpu.BindGroupLayout.
func (pc *Pipeline) BindGroupLayout(idx uint32) *wgpu.BindGroupLayout {
	if !pc.bindGroupLayouts[idx].IsValid() {
		panic("BindGroupLayout not valid anymore")
	}

	return pc.bindGroupLayouts[idx]
}

func releasePipelineOnEvict[C any](_ C, pipe Pipeline) {
	pipe.pipeline.Release()
}

// PipelineCache caches render pipelines & bind group layout
type PipelineCache struct {
	ctx                  *RenderContext
	bindGroupLayoutCache []cachedBindGroupLayout
	preCompiler          *pre.Compiler
}

//goland:noinspection GoMixedReceiverTypes
func (PipelineCache) FromWorld(world *byke.World) PipelineCache {
	return PipelineCache{
		ctx:         byke.RequireResourceOf[RenderContext](world),
		preCompiler: byke.RequireResourceOf[pre.Compiler](world),
	}
}

func (p *PipelineCache) Shader(label, shaderCode string, values ShaderValues) *wgpu.ShaderModule {
	shaderCode, err := p.preCompiler.PreCompile(shaderCode, values)
	if err != nil {
		panic(fmt.Errorf("prepare shader %q: %w", label, err))
	}

	return p.ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      label,
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: shaderCode},
	})
}

func (p *PipelineCache) BindGroupLayout(desc wgpu.BindGroupLayoutDescriptor) *wgpu.BindGroupLayout {
	for _, cached := range p.bindGroupLayoutCache {
		if cached.Matches(desc) {
			return cached.BindGroupLayout
		}
	}

	slog.Debug("Create BindGroupLayout", slog.String("label", desc.Label))
	bindGroupLayout := p.ctx.CreateBindGroupLayout(new(desc))

	p.bindGroupLayoutCache = append(p.bindGroupLayoutCache, cachedBindGroupLayout{
		Descriptor:      desc,
		BindGroupLayout: bindGroupLayout,
	})

	return bindGroupLayout
}

type cachedBindGroupLayout struct {
	Descriptor      wgpu.BindGroupLayoutDescriptor
	BindGroupLayout *wgpu.BindGroupLayout
}

func (c *cachedBindGroupLayout) Matches(desc wgpu.BindGroupLayoutDescriptor) bool {
	return desc.Label == c.Descriptor.Label && slices.Equal(desc.Entries, c.Descriptor.Entries)
}
