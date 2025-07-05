package byke

type scheduleId struct {
	// make sure this is not a zero-sized type
	_ int32
}

var (
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
)
