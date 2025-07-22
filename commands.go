package byke

import (
	"github.com/oliverbestmann/byke/spoke"
	"reflect"
)

type Command func(world *World)

type EntityCommand func(world *World, entityId EntityId)

// Commands is a SystemParam that allows you to send commands to a world.
// It allows you to spawn and despawn entities and to add and remove components.
// It must be injected as a pointer into a system.
type Commands struct {
	world *World
	queue []Command
}

func (c *Commands) applyToWorld() {
	for _, command := range c.queue {
		command(c.world)
	}

	// reset the queue after applying it
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

func (c *Commands) Spawn(components ...ErasedComponent) EntityCommands {
	entityId := c.world.reserveEntityId()

	c.Queue(func(world *World) {
		world.spawnWithEntityId(entityId, components)
	})

	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
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
	e.commands.queue = append(e.commands.queue, func(world *World) {
		for _, command := range commands {
			command(world, e.entityId)
		}
	})

	return e
}

func (e EntityCommands) Despawn() {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		world.Despawn(e.entityId)
	})
}

func (e EntityCommands) Observe(system AnySystem) EntityCommands {
	return e.Update(func(world *World, entityId EntityId) {
		world.AddObserver(NewObserver(system).WatchEntity(entityId))
	})
}

func (e EntityCommands) Trigger(eventValue any) EntityCommands {
	return e.Update(func(world *World, entityId EntityId) {
		world.TriggerObserver(entityId, eventValue)
	})
}

func RemoveComponent[C IsComponent[C]]() EntityCommand {
	componentType := spoke.ComponentTypeOf[C]()

	return func(world *World, entityId EntityId) {
		world.removeComponent(entityId, componentType)
	}
}

func InsertComponent[C IsComponent[C]](maybeValue ...C) EntityCommand {
	if len(maybeValue) > 1 {
		panic("InsertComponent must be called with at most one argument")
	}

	var component C
	if len(maybeValue) == 1 {
		component = maybeValue[0]
	}

	return func(world *World, entityId EntityId) {
		world.insertComponents(entityId, []ErasedComponent{component})
	}
}

type commandSystemParamState Commands

func (c *commandSystemParamState) getValue(systemContext) reflect.Value {
	return reflect.ValueOf((*Commands)(c))
}

func (c *commandSystemParamState) cleanupValue() {
	(*Commands)(c).applyToWorld()
}

func (*commandSystemParamState) valueType() reflect.Type {
	return reflect.TypeFor[*Commands]()
}
