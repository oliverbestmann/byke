package byke

import (
	"time"
)

type Timings struct {
	Count         int
	Latest        time.Duration
	MovingAverage time.Duration
	Min, Max      time.Duration
}

func (t Timings) Add(d time.Duration) Timings {
	t.Latest = d

	if t.Count == 0 {
		t.Min = d
		t.Max = d
	} else {
		t.Min = min(t.Min, d)
		t.Max = max(t.Max, d)
	}

	t.MovingAverage = (95*t.MovingAverage + 5*d) / 100

	t.Count += 1

	return t
}

type TimingStats struct {
	Visible bool

	BySchedule    map[ScheduleId]Timings
	ScheduleOrder []ScheduleId

	BySystem map[*preparedSystem]Timings
}

func NewTimingStats() TimingStats {
	return TimingStats{
		BySchedule: map[ScheduleId]Timings{},
		BySystem:   map[*preparedSystem]Timings{},
	}
}

func (t *TimingStats) MeasureSchedule(scheduleId ScheduleId) TimingStopwatch {
	startTime := time.Now()

	if _, ok := t.BySchedule[scheduleId]; !ok {
		t.ScheduleOrder = append(t.ScheduleOrder, scheduleId)
	}

	return TimingStopwatch{
		Stop: func() {
			duration := time.Since(startTime)
			t.BySchedule[scheduleId] = t.BySchedule[scheduleId].Add(duration)
		},
	}
}

func (t *TimingStats) MeasureSystem(system *preparedSystem) TimingStopwatch {
	startTime := time.Now()

	return TimingStopwatch{
		Stop: func() {
			duration := time.Since(startTime)
			t.BySystem[system] = t.BySystem[system].Add(duration)
		},
	}
}

type TimingStopwatch struct {
	Stop func()
}
