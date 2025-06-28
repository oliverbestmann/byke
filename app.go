package ecs

var RunWorld = &Schedule{}

type App struct {
	world *World
}

type Plugin func(app *App)

func (a *App) World() *World {
	if a.world == nil {
		world := NewWorld()
		a.world = &world
	}

	return a.world
}

func (a *App) AddPlugin(plugin Plugin) {
	plugin(a)
}

func (a *App) AddSystems(schedule *Schedule, systems ...System) {
	a.World().AddSystems(schedule, systems...)
}

func (a *App) InsertResource(res any) {
	a.World().InsertResource(res)
}

func (a *App) Run() {
	a.World().RunSchedule(RunWorld)
}
