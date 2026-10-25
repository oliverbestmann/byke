//byke:no wasm

package main

import (
	"fmt"

	. "github.com/oliverbestmann/byke"
)

// Systems talk to each other with messages. One system writes them, any number of
// systems read them. Each reader sees every message once, within two frames.
// The example prints to the console and exits after a few frames.

type Damage struct {
	Target string
	Amount int
}

type Log struct {
	Text string
}

func main() {
	var app App

	// every message type must be registered first
	app.AddMessage[Damage]()
	app.AddMessage[Log]()

	app.AddSystems(Update, System(attackSystem, applyDamageSystem, printLogSystem).Chain())

	// a reader running before the writer reads the messages of the previous frame
	app.AddSystems(PreUpdate, countDamageSystem)

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

func attackSystem(damage *MessageWriter[Damage], frame *Local[int]) {
	frame.Value++

	damage.Write(Damage{Target: "Goblin", Amount: 5 * frame.Value})
	damage.Write(Damage{Target: "Troll", Amount: 3 * frame.Value})
}

func applyDamageSystem(damage *MessageReader[Damage], log *MessageWriter[Log]) {
	for _, msg := range damage.Read() {
		log.Write(Log{Text: fmt.Sprintf("%s takes %d damage", msg.Target, msg.Amount)})
	}
}

func printLogSystem(log *MessageReader[Log]) {
	for _, msg := range log.Read() {
		fmt.Println(" ", msg.Text)
	}
}

func countDamageSystem(damage *MessageReader[Damage], total *Local[int]) {
	for _, msg := range damage.Read() {
		total.Value += msg.Amount
	}

	fmt.Printf("  Total damage so far: %d\n", total.Value)
}
