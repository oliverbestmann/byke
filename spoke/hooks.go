package spoke

import (
	"fmt"
)

type ComponentHook func(entity EntityRef, component *ComponentType)

type ComponentHooks struct {
	OnAdd     ComponentHook
	OnInsert  ComponentHook
	OnDiscard ComponentHook
	OnRemove  ComponentHook
	OnDespawn ComponentHook
}

type RegisterComponentHooks struct {
	storage       *Storage
	componentType *ComponentType
	hooks         ComponentHooks
}

func (r *RegisterComponentHooks) OnAdd(hook ComponentHook) *RegisterComponentHooks {
	if r.hooks.OnAdd != nil {
		panic(fmt.Errorf("component hook OnAdd already set for %q", r.componentType))
	}

	r.hooks.OnAdd = hook
	return r.update()
}

func (r *RegisterComponentHooks) OnInsert(hook ComponentHook) *RegisterComponentHooks {
	if r.hooks.OnInsert != nil {
		panic(fmt.Errorf("component hook OnInsert already set for %q", r.componentType))
	}

	r.hooks.OnInsert = hook
	return r.update()
}

func (r *RegisterComponentHooks) OnDiscard(hook ComponentHook) *RegisterComponentHooks {
	if r.hooks.OnDiscard != nil {
		panic(fmt.Errorf("component hook OnDiscard already set for %q", r.componentType))
	}

	r.hooks.OnDiscard = hook
	return r.update()
}

func (r *RegisterComponentHooks) OnRemove(hook ComponentHook) *RegisterComponentHooks {
	if r.hooks.OnRemove != nil {
		panic(fmt.Errorf("component hook OnRemove already set for %q", r.componentType))
	}

	r.hooks.OnRemove = hook
	return r.update()
}

func (r *RegisterComponentHooks) OnDespawn(hook ComponentHook) *RegisterComponentHooks {
	if r.hooks.OnDespawn != nil {
		panic(fmt.Errorf("component hook OnDespawn already set for %q", r.componentType))
	}

	r.hooks.OnDespawn = hook
	return r.update()
}

func (r *RegisterComponentHooks) update() *RegisterComponentHooks {
	r.storage.hooks[r.componentType.Id] = r.hooks
	return r
}
