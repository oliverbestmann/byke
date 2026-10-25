//byke:no wasm

package main

import (
	"fmt"

	. "github.com/oliverbestmann/byke"
)

// Triggers run observers as soon as the commands of the triggering system are applied,
// unlike messages, which wait for a reader system.
// An entity observer only sees events targeted at its entity, a global observer
// sees all events of a type.
// The example prints to the console and exits after a few frames.

type Fuse struct {
	Component[Fuse]
	Frames int
}

// Explode targets a single entity. Embedding EventTarget makes it an EntityEvent.
type Explode struct {
	EventTarget
}

func main() {
	var app App

	app.World().AddObserver(NewObserver(countExplosionsObserver))

	app.AddSystems(Startup, spawnSystem)
	app.AddSystems(Update, burnFuseSystem)

	// run two frames, then exit
	app.RunWorld(func(world *World) error {
		for frame := range 2 {
			fmt.Printf("\nFrame %d\n", frame)
			world.RunSchedule(Main)
		}

		return nil
	})

	app.MustRun()
}

func spawnSystem(commands *Commands) {
	for idx, frames := range []int{1, 2, 2} {
		commands.
			Spawn(Named(fmt.Sprintf("Bomb %d", idx)), Fuse{Frames: frames}).
			Observe(explodeObserver)
	}
}

func burnFuseSystem(commands *Commands, query Query[struct {
	EntityId
	Name Name
	Fuse *Fuse
}],
) {
	for item := range query.Items() {
		item.Fuse.Frames--
		fmt.Printf("  %s burns, fuse at %d\n", item.Name, item.Fuse.Frames)

		if item.Fuse.Frames == 0 {
			commands.Entity(item.EntityId).Trigger(Explode{EventTarget: EventTarget(item.EntityId)})
		}
	}
}

// The first parameter of an observer is the event, other parameters work like in systems.
func explodeObserver(trigger On[Explode], commands *Commands, names Query[Name]) {
	id := trigger.Event.TargetEntityId()

	name, _ := names.Get(id)
	fmt.Printf("  %s explodes\n", name)

	commands.Entity(id).Despawn()
}

// A global observer runs for the Explode events of every entity.
func countExplosionsObserver(trigger On[Explode], count *Local[int]) {
	count.Value++
	fmt.Printf("  That was explosion %d\n", count.Value)
}
