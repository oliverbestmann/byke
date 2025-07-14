package bykebiten

import (
	"math"
	"time"
)

type TimerMode uint8

const TimerModeOnce TimerMode = 0
const TimerModeRepeating TimerMode = 1

type Timer struct {
	duration time.Duration
	elapsed  time.Duration

	finishedCountInTick uint32
	finished            bool
	mode                TimerMode
}

func NewTimer(duration time.Duration, mode TimerMode) Timer {
	return Timer{
		duration: duration,
		mode:     mode,
	}
}

func NewTimerFromSeconds(seconds float64, mode TimerMode) Timer {
	duration := time.Duration(seconds * float64(time.Second))
	return Timer{
		duration: duration,
		mode:     mode,
	}
}

func (t *Timer) Tick(delta time.Duration) *Timer {
	t.finishedCountInTick = 0

	if t.finished && t.mode == TimerModeOnce {
		// nothing to do, timer is done
		return t
	}

	t.elapsed += delta

	if t.elapsed >= t.duration && t.duration > 0 {
		if t.mode == TimerModeOnce {
			// normal timer will stop here
			t.elapsed = t.duration
			t.finished = true
			t.finishedCountInTick = 1
			return t
		}

		if t.mode == TimerModeRepeating {
			t.finishedCountInTick = uint32(min(math.MaxUint32, t.elapsed/t.duration))

			// repeating timer resets elapsed time
			t.elapsed = t.elapsed % t.duration
		}
	}

	return t
}

func (t *Timer) Duration() time.Duration {
	return t.duration
}

func (t *Timer) Elapsed() time.Duration {
	return t.elapsed
}

func (t *Timer) Remaining() time.Duration {
	return t.duration - t.elapsed
}

func (t *Timer) Fraction() float64 {
	return float64(t.elapsed) / float64(t.duration)
}

func (t *Timer) FractionRemaining() float64 {
	return 1 - t.Fraction()
}

func (t *Timer) Finished() bool {
	return t.finished
}

func (t *Timer) JustFinished() bool {
	return t.finishedCountInTick > 0
}

func (t *Timer) TimesFinishedThisTick() int {
	return int(t.finishedCountInTick)
}
func (t *Timer) Reset() {
	t.elapsed = 0
	t.finished = false
	t.finishedCountInTick = 0
}
