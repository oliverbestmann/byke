package byke

import (
	"math"
	"time"
)

type TimerMode uint8

const TimerModeOnce TimerMode = 0
const TimerModeRepeating TimerMode = 1

// Timer is either a one of or a repeating timer with a specific duration.
type Timer struct {
	duration time.Duration
	elapsed  time.Duration

	finishedCountInTick uint32
	finished            bool
	mode                TimerMode
}

// NewTimer creates a new timer
func NewTimer(duration time.Duration, mode TimerMode) Timer {
	return Timer{
		duration: duration,
		mode:     mode,
	}
}

// Tick adds the given amount of time to the Timer.
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

// Duration returns the configured duration of the Timer.
func (t *Timer) Duration() time.Duration {
	return t.duration
}

// Elapsed returns the already elapsed time of the Timer.
func (t *Timer) Elapsed() time.Duration {
	return t.elapsed
}

// Remaining returns the remaining time of the Timer.
func (t *Timer) Remaining() time.Duration {
	return t.duration - t.elapsed
}

// Fraction returns the fraction to that this timer has finished. A freshly started timer
// will have a Fraction value of 0.
func (t *Timer) Fraction() float64 {
	return float64(t.elapsed) / float64(t.duration)
}

// FractionRemaining is the inverse of Fraction.
func (t *Timer) FractionRemaining() float64 {
	return 1 - t.Fraction()
}

// Finished returns true if the timer has finished.
// In case this is a timer with mode TimerModeRepeating, this method will never return true.
func (t *Timer) Finished() bool {
	return t.finished
}

// JustFinished returns true if the timer has reached its duration at the previous call to Tick.
func (t *Timer) JustFinished() bool {
	return t.finishedCountInTick > 0
}

// TimesFinishedThisTick returns the number of times this timer has finished at the previous call to Tick.
// E.g. if you tick a 1 second timer with a 3.5 second delta, the timer will have finished three times in this tick.
func (t *Timer) TimesFinishedThisTick() int {
	return int(t.finishedCountInTick)
}

// Reset resets the timer back to its starting point.
func (t *Timer) Reset() {
	t.elapsed = 0
	t.finished = false
	t.finishedCountInTick = 0
}
