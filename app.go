package byke

import (
	"fmt"
	"reflect"
)

type App struct {
	world *World
	run   RunWorld
}

func (a *App) World() *World {
	if a.world == nil {
		a.world = NewWorld()

		configureSchedules(a)
	}

	return a.world
}

func (a *App) AddPlugin(plugin Plugin) {
	plugin.ApplyTo(a)
}

func (a *App) AddSystems(scheduleId ScheduleId, system AnySystem, systems ...AnySystem) {
	if !reflect.ValueOf(scheduleId).Comparable() {
		panic(fmt.Sprintf("scheduleId must be comparable: %C", scheduleId))
	}

	a.World().AddSystems(scheduleId, system, systems...)
}

func (a *App) InsertResource(res any) {
	a.World().InsertResource(res)
}

func (a *App) InitState(newState NewState) {
	newState.configureStateIn(a)
}

func (a *App) RegisterEvent(newEvent NewEventType) {
	newEvent.configureEventIn(a)
}

func (a *App) RunWorld(run RunWorld) {
	a.run = run
}

func (a *App) Run() error {
	if a.run == nil {
		a.run = func(world *World) error {
			for {
				world.RunSchedule(Main)
			}
		}
	}

	return a.run(a.World())
}

type Plugin interface {
	ApplyTo(app *App)
}

type PluginFunc func(app *App)

func (plugin PluginFunc) ApplyTo(app *App) {
	plugin(app)
}

type RunWorld func(world *World) error

type NewState interface {
	configureStateIn(app *App)
}
