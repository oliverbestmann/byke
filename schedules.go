package byke

import (
	"time"
)

type scheduleId struct {
	// make sure this is not a zero-sized type
	_ int32
}

func (*scheduleId) isSchedule() {}

var (
	Main             ScheduleId = &scheduleId{}
	RunFixedMainLoop ScheduleId = &scheduleId{}
	FixedMain        ScheduleId = &scheduleId{}

	PreStartup      ScheduleId = &scheduleId{}
	Startup         ScheduleId = &scheduleId{}
	PostStartup     ScheduleId = &scheduleId{}
	First           ScheduleId = &scheduleId{}
	PreUpdate       ScheduleId = &scheduleId{}
	StateTransition ScheduleId = &scheduleId{}
	Update          ScheduleId = &scheduleId{}
	PostUpdate      ScheduleId = &scheduleId{}
	PreRender       ScheduleId = &scheduleId{}
	Render          ScheduleId = &scheduleId{}
	PostRender      ScheduleId = &scheduleId{}
	Last            ScheduleId = &scheduleId{}

	FixedFirst      ScheduleId = &scheduleId{}
	FixedPreUpdate  ScheduleId = &scheduleId{}
	FixedUpdate     ScheduleId = &scheduleId{}
	FixedPostUpdate ScheduleId = &scheduleId{}
	FixedLast       ScheduleId = &scheduleId{}
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
