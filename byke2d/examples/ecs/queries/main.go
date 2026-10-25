//byke:no wasm

package main

import (
	"fmt"

	. "github.com/oliverbestmann/byke"
)

// Shows the query features: plain iteration, mutable access, the With, Without,
// Added and Changed filters, optional components, Single and lookup by EntityId.
// The example prints to the console and exits after a few frames.

type Health struct {
	// a comparable component supports change detection with Changed
	ComparableComponent[Health]
	Value int
}

type Poisoned struct {
	ImmutableComponent[Poisoned]
}

type Shield struct {
	ImmutableComponent[Shield]
	Strength int
}

type Player struct {
	ImmutableComponent[Player]
}

func main() {
	var app App

	app.AddSystems(Startup, spawnSystem)
	app.AddSystems(Update, System(
		addedSystem,
		poisonSystem,
		changedSystem,
		shieldSystem,
		playerSystem,
	).Chain())

	// run three frames, then exit
	app.RunWorld(func(world *World) error {
		for frame := range 3 {
			fmt.Printf("\nFrame %d\n", frame)
			world.RunSchedule(Main)
		}

		return nil
	})

	app.MustRun()
}

func spawnSystem(commands *Commands) {
	commands.Spawn(Player{}, Named("Hero"), Health{Value: 100}, Shield{Strength: 5})
	commands.Spawn(Named("Goblin"), Health{Value: 30}, Poisoned{})
	commands.Spawn(Named("Troll"), Health{Value: 80}, Poisoned{}, Shield{Strength: 2})
	commands.Spawn(Named("Rock"))
}

// Added matches only entities whose Health was added since the system last ran.
// It prints in the first frame only.
func addedSystem(query Query[struct {
	_      Added[Health]
	Name   Name
	Health Health
}],
) {
	for item := range query.Items() {
		fmt.Printf("  %s spawned with %d health\n", item.Name, item.Health.Value)
	}
}

// A pointer field gives mutable access. With filters for entities that have
// a component, without fetching it.
func poisonSystem(query Query[struct {
	_      With[Poisoned]
	Health *Health
}],
) {
	for item := range query.Items() {
		item.Health.Value -= 10
	}
}

// Changed matches only entities whose Health value differs from the last run.
// A newly added component counts as changed.
func changedSystem(query Query[struct {
	_      Changed[Health]
	Name   Name
	Health Health
}],
) {
	for item := range query.Items() {
		fmt.Printf("  %s now has %d health\n", item.Name, item.Health.Value)
	}
}

// Option fetches a component if the entity has it. Without excludes the player.
func shieldSystem(query Query[struct {
	_      Without[Player]
	Name   Name
	Shield Option[Shield]
}],
) {
	for item := range query.Items() {
		if shield, ok := item.Shield.Get(); ok {
			fmt.Printf("  %s has a shield of strength %d\n", item.Name, shield.Strength)
		} else {
			fmt.Printf("  %s has no shield\n", item.Name)
		}
	}
}

// Single fetches the only matching entity. The system is skipped if there is
// none or more than one. Query.Get looks up a single entity by its id.
func playerSystem(
	player Single[struct {
		EntityId
		_ With[Player]
	}],
	query Query[struct {
		Name   Name
		Health Health
	}],
) {
	if item, ok := query.Get(player.Value.EntityId); ok {
		fmt.Printf("  The player is %s with %d health\n", item.Name, item.Health.Value)
	}
}
