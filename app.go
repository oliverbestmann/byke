package ecs

type App struct {
	world *World
	run   RunWorld
}

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

func (a *App) RunWorld(run RunWorld) {
	a.run = run
}

func (a *App) Run() error {
	return a.run(a.World())
}

type Plugin func(app *App)

type RunWorld func(world *World) error
