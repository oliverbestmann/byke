package byke2d

import (
	"errors"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/internal/refl"
	"github.com/oliverbestmann/pulse/wx"
)

type PipelineConfig = wx.PipelineConfig

type Pipelines[C PipelineConfig] struct {
	cache *wx.PipelineCache[C]
}

func (p Pipelines[C]) Specialize(config C) wx.CachedPipeline {
	return p.cache.Get(config)
}

func (Pipelines[C]) newState(world *byke.World, _ pipelinesT) byke.SystemParamState {
	return &pipelineCacheSystemParamState[C]{World: world}
}

type pipelinesT interface {
	newState(world *byke.World, _ pipelinesT) byke.SystemParamState
}

func makePipelinesSystemParamState(world *byke.World, pType reflect.Type) byke.SystemParamState {
	if !refl.ImplementsInterfaceDirectly[pipelinesT](pType) {
		return nil
	}

	// pType is Pipelines[C]
	p := reflect.New(pType).Elem().Interface().(pipelinesT)
	return p.newState(world, p)
}

type pipelineCacheSystemParamState[C PipelineConfig] struct {
	World    *byke.World
	instance reflect.Value
}

func (p *pipelineCacheSystemParamState[C]) GetValue(byke.SystemContext) (reflect.Value, error) {
	if !p.instance.IsValid() {
		ctx, ok := byke.ResourceOf[RenderContext](p.World)
		if !ok {
			return reflect.Value{}, errors.New("no RenderContext in World")
		}

		pipelines := Pipelines[C]{cache: wx.NewPipelineCache[C](ctx.Context)}
		p.instance = reflect.ValueOf(pipelines)
	}

	return p.instance, nil
}

func (p *pipelineCacheSystemParamState[C]) ValueType() reflect.Type {
	return reflect.TypeFor[Pipelines[C]]()
}

func (p *pipelineCacheSystemParamState[C]) CleanupValue() {
}
