package byke2d

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/meh"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type ShaderValues = pre.Values

type PipelineContext interface {
	// Shader compiles a shader to a *wgpu.ShaderModule.
	Shader(label, shaderCode string, values ShaderValues) *wgpu.ShaderModule
}

type PipelineConfig interface {
	Specialize(ctx PipelineContext) RenderPipelineDescriptor
	EqualTo(other PipelineConfig) bool
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

// PipelineCache caches render pipelines & bind group layout
type PipelineCache struct {
	ctx                  *RenderContext
	bindGroupLayoutCache []cachedBindGroupLayout
	preCompiler          *pre.Compiler
	pipelines            meh.Map[PipelineConfig, Pipeline]
}

func (p *PipelineCache) Specialize(config PipelineConfig) Pipeline {
	cached, ok := p.pipelines.Get(config)
	if ok {
		return cached
	}

	// create the pipeline descriptor
	desc := config.Specialize(p)

	// create bind group layouts
	var bgls []*wgpu.BindGroupLayout
	for _, bgld := range desc.Layout {
		bgl := p.BindGroupLayout(bgld)
		bgls = append(bgls, bgl)
	}

	// create the pipeline layout
	layout := p.ctx.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            desc.Label,
		BindGroupLayouts: bgls,
	})

	defer layout.Release()

	// now create the actual pipeline
	pipe := p.ctx.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:        desc.Label,
		Layout:       layout,
		Vertex:       desc.Vertex,
		Primitive:    desc.Primitive,
		DepthStencil: desc.DepthStencil,
		Multisample:  desc.Multisample,
		Fragment:     desc.Fragment,
	})

	pipeline := Pipeline{
		pipeline:         wgpu.Share(pipe),
		bindGroupLayouts: bgls,
	}

	p.pipelines.Insert(config, pipeline)

	return pipeline
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
	bindGroupLayout := wgpu.Share(p.ctx.CreateBindGroupLayout(new(desc)))

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
	return desc.Label == c.Descriptor.Label && slices.EqualFunc(desc.Entries, c.Descriptor.Entries, func(lhs, rhs wgpu.BindGroupLayoutEntry) bool {
		return lhs.Binding == rhs.Binding &&
			lhs.Visibility == rhs.Visibility &&
			lhs.Buffer == rhs.Buffer &&
			lhs.Sampler == rhs.Sampler &&
			lhs.Texture == rhs.Texture &&
			lhs.StorageTexture == rhs.StorageTexture
	})
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
