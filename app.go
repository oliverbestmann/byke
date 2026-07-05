package byke

import (
	"fmt"
	"reflect"
)

// App provides an entry point to your application.
type App struct {
	world *World
	run   Runner
}

// World returns the world created by the App.
func (a *App) World() *World {
	if a.world == nil {
		a.world = NewWorld()

		configureSchedules(a)
	}

	return a.world
}

// AddPlugin adds the given Plugin to this app.
func (a *App) AddPlugin(plugin Plugin) {
	plugin(a)
}

// AddMakeSystemParam adds the given MakeSystemParam to this app.
func (a *App) AddMakeSystemParam(msp MakeSystemParam) {
	a.World().AddMakeSystemParam(msp)
}

// AddSystems adds one or more systems to the World.
func (a *App) AddSystems(scheduleId ScheduleId, system AnySystem, systems ...AnySystem) {
	if !reflect.ValueOf(scheduleId).Comparable() {
		panic(fmt.Errorf("scheduleId must be comparable: %T", scheduleId))
	}

	a.World().AddSystems(scheduleId, system, systems...)
}

// ConfigureSystemSets configures sets in a schedule.
func (a *App) ConfigureSystemSets(scheduleId ScheduleId, sets ...*SystemSet) {
	if !reflect.ValueOf(scheduleId).Comparable() {
		panic(fmt.Sprintf("scheduleId must be comparable: %T", scheduleId))
	}

	a.World().ConfigureSystemSets(scheduleId, sets...)
}

// InsertResource inserts a resource into the World.
// See World.InsertResource.
func (a *App) InsertResource(res any) {
	a.World().InsertResource(res)
}

// InitResource initializes a resource.
// You can provide an optional initialization function.
func (a *App) InitResource[T any](init ...InitFromWorld[T]) {
	world := a.World()

	switch len(init) {
	case 0:
		var zeroT T
		world.InsertResource(zeroT)
	case 1:
		world.InsertResource(init[0](world))
	default:
		panic("InitResource must be invoked with zero or one parameter")
	}
}

// InitState configures a new state in the World.
// Use StateType to acquire a value implementing stateType.
func (a *App) InitState[S comparable](initialValue S) {
	a.AddPlugin(pluginState(initialValue))
}

// AddMessage configures a new event in the World.
// Use MessageType to acquire a value implementing AddMessageType.
func (a *App) AddMessage[E any]() {
	a.AddPlugin(pluginMessage[E])
}

// RunWorld configures the function that is executed in Run.
// This is normally used by plugins to do custom setup like
// creating a new window and setting up the renderer.
//
// If not called, Run will simply run the Main schedule in a loop.
func (a *App) RunWorld(run Runner) {
	a.run = run
}

// Run will run the Runner configured in Runner.
func (a *App) Run() error {
	if a.run == nil {
		a.run = func(world *World) error {
			for {
				world.RunSchedule(Main)
			}
		}
	}

	a.World().PrintSystems()

	return a.run(a.World())
}

// MustRun calls Run and panics if Run returns an error.
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Plugin for an App.
// Call App.AddPlugin to add a Plugin to an App.
type Plugin func(app *App)

type Runner func(world *World) error

type InitFromWorld[T any] func(world *World) T
