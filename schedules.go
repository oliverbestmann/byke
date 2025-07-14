package byke

import (
	"time"
)

type scheduleId struct {
	name string
}

func (*scheduleId) isSchedule() {}

func (s *scheduleId) String() string {
	return s.name
}

func MakeScheduleId(name string) ScheduleId {
	return &scheduleId{name: name}
}

var (
	Main             ScheduleId = MakeScheduleId("Main")
	RunFixedMainLoop ScheduleId = MakeScheduleId("RunFixedMainLoop")
	FixedMain        ScheduleId = MakeScheduleId("FixedMain")

	PreStartup      ScheduleId = MakeScheduleId("PreStartup")
	Startup         ScheduleId = MakeScheduleId("Startup")
	PostStartup     ScheduleId = MakeScheduleId("PostStartup")
	First           ScheduleId = MakeScheduleId("First")
	PreUpdate       ScheduleId = MakeScheduleId("PreUpdate")
	StateTransition ScheduleId = MakeScheduleId("StateTransition")
	Update          ScheduleId = MakeScheduleId("Update")
	PostUpdate      ScheduleId = MakeScheduleId("PostUpdate")
	PreRender       ScheduleId = MakeScheduleId("PreRender")
	Render          ScheduleId = MakeScheduleId("Render")
	PostRender      ScheduleId = MakeScheduleId("PostRender")
	Last            ScheduleId = MakeScheduleId("Last")

	FixedFirst      ScheduleId = MakeScheduleId("FixedFirst")
	FixedPreUpdate  ScheduleId = MakeScheduleId("FixedPreUpdate")
	FixedUpdate     ScheduleId = MakeScheduleId("FixedUpdate")
	FixedPostUpdate ScheduleId = MakeScheduleId("FixedPostUpdate")
	FixedLast       ScheduleId = MakeScheduleId("FixedLast")
)

func configureSchedules(app *App) {
	app.InsertResource(VirtualTime{
		Scale: 1.0,
	})

	app.InsertResource(FixedTime{
		// 64 hz, same as bevy
		StepInterval: 1 * time.Second / 64,
	})

	app.AddSystems(Main, System(updateVirtualTime, runMainSchedule).Chain())
	app.AddSystems(RunFixedMainLoop, runFixedMainLoopSystem)
	app.AddSystems(FixedMain, runFixedMainScheduleSystem)
}

func runMainSchedule(world *World, initialized *Local[bool]) {
	if !initialized.Value {
		initialized.Value = true

		// initialize once
		world.RunSchedule(PreStartup)
		world.RunSchedule(StateTransition)
		world.RunSchedule(Startup)
		world.RunSchedule(PostStartup)
	}

	// start the new frame
	world.RunSchedule(First)

	// the update schedule
	world.RunSchedule(PreUpdate)
	world.RunSchedule(StateTransition)
	world.RunSchedule(RunFixedMainLoop)
	world.RunSchedule(Update)
	world.RunSchedule(PostUpdate)

	world.RunSchedule(PreRender)
	world.RunSchedule(Render)
	world.RunSchedule(PostRender)

	// end the frame
	world.RunSchedule(Last)
}

func runFixedMainLoopSystem(world *World, ft *FixedTime, vt VirtualTime) {
	ft.overstep += vt.Delta

	step := ft.StepInterval

	for ft.overstep >= step {
		ft.overstep -= step

		ft.Elapsed += step
		ft.Delta = step
		ft.DeltaSecs = step.Seconds()

		world.RunSchedule(FixedMain)
	}
}

func runFixedMainScheduleSystem(world *World) {
	world.RunSchedule(FixedFirst)
	world.RunSchedule(FixedPreUpdate)
	world.RunSchedule(FixedUpdate)
	world.RunSchedule(FixedPostUpdate)
	world.RunSchedule(FixedLast)
}
