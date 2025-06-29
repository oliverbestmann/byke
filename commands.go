package byke

import "fmt"

type Command func(world *World)

type EntityCommand func(world *World, entity *Entity)

type Commands struct {
	world *World
	queue []Command
}

func (c *Commands) Queue(command Command) *Commands {
	c.queue = append(c.queue, command)
	return c
}

func (c *Commands) Spawn(components ...AnyComponent) EntityCommands {
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

func (e EntityCommands) Update(commands ...EntityCommand) EntityCommands {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		entity, ok := world.entities[e.entityId]
		if !ok {
			panic(fmt.Sprintf("entity %d does not exist", e.entityId))
		}

		for _, command := range commands {
			command(world, entity)
		}
	})

	return e
}

func (e EntityCommands) Despawn() {
	e.commands.queue = append(e.commands.queue, func(world *World) {
		if world.entities[e.entityId] == nil {
			fmt.Printf("[warn] cannot despawn entity %d: does not exist\n", e.entityId)
			return
		}

		delete(world.entities, e.entityId)
	})
}

func RemoveComponent[C IsComponent[C]]() EntityCommand {
	componentType := componentTypeOf[C]()

	return func(world *World, entity *Entity) {
		delete(entity.Components, componentType)
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

	return func(world *World, entity *Entity) {
		world.insertComponents(entity, []AnyComponent{component})
	}
}
