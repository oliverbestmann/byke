package byke

import (
	"fmt"
	"reflect"

	"github.com/oliverbestmann/byke/spoke"
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

// Add adds a command to be executed.
func (c *Commands) Add(command Command) *Commands {
	if len(c.queue) > 0 {
		lastIndex := len(c.queue) - 1
		if prev, ok := c.queue[lastIndex].(mergeableCommand); ok {
			if prev.MergeWith(command) {
				return c
			}
		}
	}

	c.queue = append(c.queue, command)
	return c
}

func (c *Commands) InsertResource(resource any) *Commands {
	return c.Add(&insertResourceCommand{Resource: resource})
}

func (c *Commands) Spawn(components ...ErasedComponent) EntityCommands {
	entityId := c.world.reserveEntityId()

	c.Add(&spawnCommand{
		EntityId:   entityId,
		Components: components,
	})

	return EntityCommands{
		entityId: entityId,
		commands: c,
	}
}

func (c *Commands) RunSystem(system AnySystem) *Commands {
	return c.Add(&runSystemCommand{
		System: system,
	})
}

func (c *Commands) RunSystemWith(system AnySystem, inValue any) *Commands {
	return c.Add(&runSystemCommand{
		System:  system,
		InValue: inValue,
	})
}

func (c *Commands) Trigger(eventValue any) *Commands {
	return c.Add(&triggerCommand{
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

func (e EntityCommands) Despawn() {
	e.commands.queue = append(e.commands.queue, &despawnCommand{e.entityId})
}

// Trigger triggers the given EntityEvent.
// TODO This should take a "func(EventId) EntityEvent" parameter to match bevy api
func (e EntityCommands) Trigger(eventValue EntityEvent) *Commands {
	if eventValue.TargetEntityId() != e.entityId {
		panic(fmt.Sprintf("EntityId %q missmatch with event: %q", e.entityId, eventValue.TargetEntityId()))
	}

	return e.commands.Add(&triggerCommand{
		EventValue: eventValue,
	})
}

func (e EntityCommands) Observe(system AnySystem) EntityCommands {
	e.commands.Add(&observeCommand{
		EntityId: e.entityId,
		System:   system,
	})

	return e
}

func (e EntityCommands) Insert(components ...ErasedComponent) EntityCommands {
	e.commands.Add(&insertComponentsCommand{
		EntityId:   e.entityId,
		Components: components,
	})

	return e
}

func (e EntityCommands) Remove[C IsComponent[C]]() EntityCommands {
	e.commands.Add(&applyEntityCommands{
		EntityId: e.entityId,
		Commands: []EntityCommand{
			(*removeComponentEntityCommand)(
				spoke.ComponentTypeOf[C](),
			),
		},
	})

	return e
}

type mergeableCommand interface {
	MergeWith(next Command) bool
}

type insertResourceCommand struct {
	Resource any
}

func (c *insertResourceCommand) Apply(world *World) {
	world.InsertResource(c.Resource)
}

type spawnCommand struct {
	EntityId   EntityId
	Components []ErasedComponent
}

func (c *spawnCommand) Apply(world *World) {
	world.spawnWithEntityId(c.EntityId, c.Components)
}

type applyEntityCommands struct {
	EntityId EntityId
	Commands []EntityCommand
}

func (c *applyEntityCommands) Apply(world *World) {
	for _, command := range c.Commands {
		command.Apply(world, c.EntityId)
	}
}

type despawnCommand struct {
	EntityId EntityId
}

func (c *despawnCommand) Apply(world *World) {
	world.Despawn(c.EntityId)
}

type observeCommand struct {
	EntityId EntityId
	System   AnySystem
}

func (c *observeCommand) Apply(world *World) {
	world.AddObserver(NewObserver(c.System).WatchEntity(c.EntityId))
}

type triggerCommand struct {
	EventValue Event
}

func (c *triggerCommand) Apply(world *World) {
	world.TriggerObserver(c.EventValue)
}

type removeComponentEntityCommand spoke.ComponentType

func (c *removeComponentEntityCommand) Apply(world *World, entityId EntityId) {
	world.removeComponent(entityId, (*spoke.ComponentType)(c))
}

type insertComponentsCommand struct {
	EntityId   EntityId
	Components []ErasedComponent
}

func (c *insertComponentsCommand) MergeWith(next Command) bool {
	switch next := next.(type) {
	case *insertComponentsCommand:
		if c.EntityId == next.EntityId {
			c.Components = append(c.Components, next.Components...)
			return true
		}

	case *spawnCommand:
		if c.EntityId == next.EntityId {
			c.Components = append(c.Components, next.Components...)
			return true
		}
	}

	return false
}

func (c *insertComponentsCommand) Apply(world *World) {
	world.insertComponents(c.EntityId, c.Components)
}

type runSystemCommand struct {
	System  AnySystem
	InValue any
}

func (c *runSystemCommand) Apply(world *World) {
	world.RunSystemWithInValue(c.System, c.InValue)
}

type commandSystemParamState Commands

func makeCommandsSystemStateParam(world *World, pType reflect.Type) SystemParamState {
	if pType != reflect.TypeFor[*Commands]() {
		return nil
	}

	return (*commandSystemParamState)(
		&Commands{world: world},
	)
}
func (c *commandSystemParamState) GetValue(SystemContext) (reflect.Value, error) {
	return reflect.ValueOf((*Commands)(c)), nil
}

func (c *commandSystemParamState) CleanupValue() {
	if len(c.queue) > 0 {
		c.world.flushes = append(c.world.flushes, func() {
			(*Commands)(c).applyToWorld()
		})
	}
}

func (*commandSystemParamState) ValueType() reflect.Type {
	return reflect.TypeFor[*Commands]()
}
