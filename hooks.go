package byke

import (
	"github.com/oliverbestmann/byke/spoke"
)

type DeferredWorld interface {
	// TODO
	Commands() *CommandQueue
}

type ComponentHook func(world DeferredWorld, entity EntityRef, componentType *ComponentType)

type ComponentHooks struct {
	OnAdd     ComponentHook
	OnInsert  ComponentHook
	OnDiscard ComponentHook
	OnRemove  ComponentHook
	OnDespawn ComponentHook
}

type RegisterComponentHooks struct {
	world    *World
	delegate *spoke.RegisterComponentHooks
}

func (r RegisterComponentHooks) OnAdd(hook ComponentHook) RegisterComponentHooks {
	_ = r.delegate.OnAdd(func(entity spoke.EntityRef, component *spoke.ComponentType) {
		r.world.dispatchHookWithWorld(entity, component, hook)
	})

	return r
}

func (r RegisterComponentHooks) OnInsert(hook ComponentHook) RegisterComponentHooks {
	_ = r.delegate.OnInsert(func(entity spoke.EntityRef, component *spoke.ComponentType) {
		r.world.dispatchHookWithWorld(entity, component, hook)
	})

	return r
}

func (r RegisterComponentHooks) OnDiscard(hook ComponentHook) RegisterComponentHooks {
	_ = r.delegate.OnDiscard(func(entity spoke.EntityRef, component *spoke.ComponentType) {
		r.world.dispatchHookWithWorld(entity, component, hook)
	})

	return r
}

func (r RegisterComponentHooks) OnRemove(hook ComponentHook) RegisterComponentHooks {
	_ = r.delegate.OnRemove(func(entity spoke.EntityRef, component *spoke.ComponentType) {
		r.world.dispatchHookWithWorld(entity, component, hook)
	})

	return r
}

func (r RegisterComponentHooks) OnDespawn(hook ComponentHook) RegisterComponentHooks {
	_ = r.delegate.OnDespawn(func(entity spoke.EntityRef, component *spoke.ComponentType) {
		r.world.dispatchHookWithWorld(entity, component, hook)
	})

	return r
}
