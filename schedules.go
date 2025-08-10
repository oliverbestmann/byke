package byke

import (
	"fmt"
	"time"
)

// ScheduleId identifies a schedule. All implementing types must be comparable.
type ScheduleId interface {
	fmt.Stringer
	isSchedule()
}

type scheduleId struct {
	name string
}

func (*scheduleId) isSchedule() {}

func (s *scheduleId) String() string {
	return s.name
}

// MakeScheduleId creates a new unique ScheduleId.
// The name passed to the schedule is used for debugging
func MakeScheduleId(name string) ScheduleId {
	return &scheduleId{name: name}
}

var (
	// Main is the main schedule that executes all other schedules in the correct order.
	Main             = MakeScheduleId("Main")
	RunFixedMainLoop = MakeScheduleId("RunFixedMainLoop")

	// FixedMain is the fixed time step main schedule that
	// executes the other fixed step schedules in the correct order.
	FixedMain = MakeScheduleId("FixedMain")

	PreStartup      = MakeScheduleId("PreStartup")
	Startup         = MakeScheduleId("Startup")
	PostStartup     = MakeScheduleId("PostStartup")
	First           = MakeScheduleId("First")
	PreUpdate       = MakeScheduleId("PreUpdate")
	StateTransition = MakeScheduleId("StateTransition")
	Update          = MakeScheduleId("Update")
	PostUpdate      = MakeScheduleId("PostUpdate")
	PreRender       = MakeScheduleId("PreRender")
	Render          = MakeScheduleId("Render")
	PostRender      = MakeScheduleId("PostRender")
	Last            = MakeScheduleId("Last")

	FixedFirst      = MakeScheduleId("FixedFirst")
	FixedPreUpdate  = MakeScheduleId("FixedPreUpdate")
	FixedUpdate     = MakeScheduleId("FixedUpdate")
	FixedPostUpdate = MakeScheduleId("FixedPostUpdate")
	FixedLast       = MakeScheduleId("FixedLast")
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
	app.AddSystems(PostUpdate, despawnWithDelaySystem)
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
