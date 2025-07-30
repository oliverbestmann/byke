package byke

import (
	"github.com/oliverbestmann/byke/spoke"
	"reflect"
)

type Command interface {
	Apply(world *World)
}

type CommandFn func(world *World)

func (c CommandFn) Apply(world *World) {
	c(world)
}

type EntityCommand interface {
	Apply(world *World, entityId EntityId)
}

// Commands is a SystemParam that allows you to send commands to a world.
// It allows you to spawn and despawn entities and to add and remove components.
// It must be injected as a pointer into a system.
type Commands struct {
	world *World
	queue []Command
}

func (c *Commands) applyToWorld() {
	for _, command := range c.queue {
		command.Apply(c.world)
	}

	// reset the queue after applying it
	clear(c.queue)

	c.queue = c.queue[:0]
}

func (*Commands) init(world *World) SystemParamState {
	return (*commandSystemParamState)(
		&Commands{world: world},
	)
}

func (c *Commands) Queue(command Command) *Commands {
	c.queue = append(c.queue, command)
	return c
}

func (c *Commands) InsertResource(resource any) *Commands {
	return c.Queue(insertResourceCommand{Resource: resource})
}

func (c *Commands) Spawn(components ...ErasedComponent) EntityCommands {
	entityId := c.world.reserveEntityId()

	c.Queue(spawnCommand{
		EntityId:   entityId,
		Components: components,
	})

	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
}

func (c *Commands) RunSystem(system AnySystem) *Commands {
	return c.Queue(runSystemCommand{
		System: system,
	})
}

func (c *Commands) RunSystemWith(system AnySystem, inValue any) *Commands {
	return c.Queue(runSystemCommand{
		System:  system,
		InValue: inValue,
	})
}

func (c *Commands) Trigger(eventValue any) *Commands {
	return c.Queue(triggerCommand{
		EntityId:   NoEntityId,
		EventValue: eventValue,
	})
}

func (c *Commands) Entity(entityId EntityId) EntityCommands {
	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
}

type EntityCommands struct {
	entityId EntityId
	commands *Commands
}

func (e EntityCommands) Id() EntityId {
	return e.entityId
}

func (e EntityCommands) Update(commands ...EntityCommand) EntityCommands {
	e.commands.Queue(applyEntityCommands{
		EntityId: e.entityId,
		Commands: commands,
	})

	return e
}

func (e EntityCommands) Despawn() {
	e.commands.queue = append(e.commands.queue, despawnCommand{e.entityId})
}

func (e EntityCommands) Observe(system AnySystem) EntityCommands {
	e.commands.Queue(observeCommand{
		EntityId: e.entityId,
		System:   system,
	})

	return e
}

func (e EntityCommands) Trigger(eventValue any) EntityCommands {
	e.commands.Queue(triggerCommand{
		EntityId:   e.entityId,
		EventValue: eventValue,
	})

	return e
}

func RemoveComponent[C IsComponent[C]]() EntityCommand {
	return (*removeComponentEntityCommand)(spoke.ComponentTypeOf[C]())
}

func InsertComponent[C IsComponent[C]](optionalValue ...C) EntityCommand {
	if len(optionalValue) > 1 {
		panic("InsertComponent must be called with at most one argument")
	}

	var component C
	if len(optionalValue) == 1 {
		component = optionalValue[0]
	}

	return insertComponentEntityCommand{
		InitialValue: component,
	}
}

type commandSystemParamState Commands

func (c *commandSystemParamState) getValue(systemContext) (reflect.Value, error) {
	return reflect.ValueOf((*Commands)(c)), nil
}

func (c *commandSystemParamState) cleanupValue() {
	if len(c.queue) > 0 {
		c.world.flushes = append(c.world.flushes, func() {
			(*Commands)(c).applyToWorld()
		})
	}
}

func (*commandSystemParamState) valueType() reflect.Type {
	return reflect.TypeFor[*Commands]()
}

type insertResourceCommand struct {
	Resource any
}

func (c insertResourceCommand) Apply(world *World) {
	world.InsertResource(c.Resource)
}

type spawnCommand struct {
	EntityId   EntityId
	Components []ErasedComponent
}

func (c spawnCommand) Apply(world *World) {
	world.spawnWithEntityId(c.EntityId, c.Components)
}

type applyEntityCommands struct {
	EntityId EntityId
	Commands []EntityCommand
}

func (c applyEntityCommands) Apply(world *World) {
	for _, command := range c.Commands {
		command.Apply(world, c.EntityId)
	}
}

type despawnCommand struct {
	EntityId EntityId
}

func (c despawnCommand) Apply(world *World) {
	world.Despawn(c.EntityId)
}

type observeCommand struct {
	EntityId EntityId
	System   AnySystem
}

func (c observeCommand) Apply(world *World) {
	world.AddObserver(NewObserver(c.System).WatchEntity(c.EntityId))
}

type triggerCommand struct {
	EntityId   EntityId
	EventValue any
}

func (c triggerCommand) Apply(world *World) {
	world.TriggerObserver(c.EntityId, c.EventValue)
}

type removeComponentEntityCommand spoke.ComponentType

func (c *removeComponentEntityCommand) Apply(world *World, entityId EntityId) {
	world.removeComponent(entityId, (*spoke.ComponentType)(c))
}

type insertComponentEntityCommand struct {
	InitialValue ErasedComponent
}

func (c insertComponentEntityCommand) Apply(world *World, entityId EntityId) {
	world.insertComponents(entityId, []ErasedComponent{c.InitialValue})
}

type runSystemCommand struct {
	System  AnySystem
	InValue any
}

func (c runSystemCommand) Apply(world *World) {
	world.RunSystemWithInValue(c.System, c.InValue)
}
