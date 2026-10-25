//byke:no wasm

package main

import (
	"fmt"

	. "github.com/oliverbestmann/byke"
)

// States switch between parts of an app, like a menu and the game. Systems can run
// on entering or leaving a state, or only while a state is active. Entities can be
// despawned automatically when their state ends.
// The example prints to the console and exits after a few frames.

type GameState int

const (
	Menu GameState = iota
	Playing
	Paused
)

func (s GameState) String() string {
	return [...]string{"Menu", "Playing", "Paused"}[s]
}

type Enemy struct {
	ImmutableComponent[Enemy]
}

func main() {
	var app App

	app.InitState(Menu)

	app.AddSystems(OnEnter(Playing), spawnEnemiesSystem)
	app.AddSystems(OnExit(Playing), func() { fmt.Println("  Leaving the game") })
	app.AddSystems(OnTransition(Paused, Playing), func() { fmt.Println("  Resuming after a pause") })

	// runs only while the game is playing
	app.AddSystems(Update, System(countEnemiesSystem).RunIf(InState(Playing)))

	app.AddSystems(Update, System(scriptSystem, logTransitionsSystem).Chain())

	// run six frames, then exit
	app.RunWorld(func(world *World) error {
		for frame := range 6 {
			fmt.Printf("\nFrame %d\n", frame)
			world.RunSchedule(Main)
		}

		return nil
	})

	app.MustRun()
}

// scriptSystem plays the role of the player. The new state applies at the
// start of the next frame.
func scriptSystem(state State[GameState], next *NextState[GameState], frame *Local[int]) {
	fmt.Printf("  State is %s\n", state.Current())

	script := map[int]GameState{0: Playing, 2: Paused, 3: Playing, 4: Menu}
	if target, ok := script[frame.Value]; ok {
		next.Set(target)
	}

	frame.Value++
}

func spawnEnemiesSystem(commands *Commands) {
	fmt.Println("  Spawning enemies")

	for range 3 {
		// despawned when the Playing state ends, also when entering Paused
		commands.Spawn(Enemy{}, DespawnOnExitState(Playing))
	}
}

func countEnemiesSystem(enemies Query[struct{ _ With[Enemy] }]) {
	fmt.Printf("  %d enemies are alive\n", enemies.Count())
}

func logTransitionsSystem(transitions *MessageReader[StateTransitionEvent[GameState]]) {
	for _, t := range transitions.Read() {
		fmt.Printf("  Transition from %s to %s\n", t.PreviousState, t.CurrentState)
	}
}
