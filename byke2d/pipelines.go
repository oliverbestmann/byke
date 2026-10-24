package byke2d

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type PipelineContext interface {
	// Shader compiles a shader to a *wgpu.ShaderModule.
	Shader(label, shaderCode string, values ShaderValues) *wgpu.ShaderModule
}

type PipelineConfig interface {
	Specialize(ctx PipelineContext) RenderPipelineDescriptor
	Hash() Hash
}

type runtimePipelineConfig struct {
	Type reflect.Type
	Hash Hash
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
	_       byke.NoCopy
	ctx     *RenderContext
	shaders *Shaders
	mutex   sync.Mutex

	pipelinesByHash map[runtimePipelineConfig]Pipeline
}

func (p *PipelineCache) Specialize[C PipelineConfig](config C) Pipeline {
	rpc := runtimePipelineConfig{
		Type: reflect.TypeFor[C](),
		Hash: config.Hash(),
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	cached, ok := p.pipelinesByHash[rpc]
	if ok {
		return cached
	}

	// create the pipeline descriptor
	desc := config.Specialize(p)

	// create bind group layouts
	var bgls []*wgpu.BindGroupLayout
	for _, bgld := range desc.Layout {
		bgl := p.ctx.CreateBindGroupLayout(bgld)
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
		config:           config,
	}

	ensureMapIsInitialized(&p.pipelinesByHash)
	p.pipelinesByHash[rpc] = pipeline

	return pipeline
}

//goland:noinspection GoMixedReceiverTypes
func PipelineCacheFromWorld(world *byke.World) PipelineCache {
	return PipelineCache{
		ctx:     world.RequireResourceOf[RenderContext](),
		shaders: world.RequireResourceOf[Shaders](),
	}
}

func (p *PipelineCache) Shader(label, shaderCode string, values ShaderValues) *wgpu.ShaderModule {
	shaderCode, err := p.shaders.Compile(label, shaderCode, values)
	if err != nil {
		panic(fmt.Errorf("prepare shader %q: %w", label, err))
	}

	return p.ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      label,
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: shaderCode},
	})
}

type Pipeline struct {
	pipeline         *wgpu.RenderPipeline
	bindGroupLayouts []*wgpu.BindGroupLayout
	config           PipelineConfig
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

// Release is a noop to satisfy the Releaser interface
func (pc Pipeline) Release() {
}
