package byke

import (
	"github.com/oliverbestmann/byke/internal/arch"
)

type Command func(world *World)

type EntityCommand func(world *World, entityId EntityId)

type Commands struct {
	world *World
	queue []Command
}

func (c *Commands) applyToWorld() {
	for _, command := range c.queue {
		command(c.world)
	}
}

func (c *Commands) Queue(command Command) *Commands {
	c.queue = append(c.queue, command)
	return c
}

func (c *Commands) Spawn(components ...ErasedComponent) EntityCommands {
	entityId := c.world.ReserveEntityId()

	c.Queue(func(world *World) {
		world.Spawn(entityId, components)
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

func RemoveComponent[C IsComponent[C]]() EntityCommand {
	componentType := arch.ComponentTypeOf[C]()

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
