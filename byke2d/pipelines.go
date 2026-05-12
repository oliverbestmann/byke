package byke2d

import (
	"fmt"
	"strings"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type ShaderValues = pre.Values

type PipelineConfig interface {
	comparable
	Specialize() SpecializedPipeline
}

type SpecializedPipeline struct {
	ShaderLabel string
	Shader      string

	// If empty, the vertex shader will be re-used for the fragment shader module
	FragmentShader string

	// will be used for the shader preprocessor
	ShaderValues ShaderValues

	// Pipeline descriptor
	Descriptor wgpu.RenderPipelineDescriptor
}

type Pipelines[C PipelineConfig] struct {
	renderContext *RenderContext
	cache         *lru.Cache[C, Pipeline]
	defines       pre.Values
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

	// create the deferred pipeline descriptor
	dp := config.Specialize()

	// compile the vertex shader, apply preprocessor if necessary
	vertexShader := p.compileShader(dp.ShaderLabel, dp.Shader, dp.ShaderValues)
	defer vertexShader.Release()

	desc := dp.Descriptor
	desc.Vertex.Module = vertexShader

	if desc.Fragment != nil {
		// re-use the vertexShader module for the fragment shader
		// if no extra fragment shader is provided
		fragmentShader := vertexShader
		if dp.FragmentShader != "" {
			fragmentShader = p.compileShader(dp.ShaderLabel, dp.FragmentShader, dp.ShaderValues)
			defer fragmentShader.Release()
		}

		desc.Fragment.Module = fragmentShader
	}

	// now create the actual pipeline
	pipe := p.renderContext.CreateRenderPipeline(&desc)

	pipeline := Pipeline{
		pipeline:    pipe,
		layoutCache: bglCache,
	}

	p.cache.Add(config, pipeline)

	return pipeline
}

func (p Pipelines[C]) compileShader(label, shaderCode string, values ShaderValues) *wgpu.ShaderModule {
	if strings.Contains(shaderCode, "#") {
		code, err := pre.Process(shaderCode, values)
		if err != nil {
			panic(fmt.Errorf("preprocessing shader %q", label))
		}
		shaderCode = code
	}

	return p.renderContext.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      label,
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: shaderCode},
	})
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
