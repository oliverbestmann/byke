package byke

import (
	"time"
)

// FixedTime should be used in fixed step systems to measure the progression of time.
//
// FixedTime will be updated using the value delta provided by VirtualTime, which might be scaled.
// In this case, time might accumulate more slowly and FixedTime steps will also be executed less often.
// To counteract this, you can decrement the StepInterval to executed fixed time step systems more often.
//
// The default value of StepInterval is taken from bevy and is 1/64s.
type FixedTime struct {
	Elapsed   time.Duration
	Delta     time.Duration
	DeltaSecs float64

	StepInterval time.Duration

	overstep time.Duration
}

// VirtualTime tracks time.
//
// The progression of time can be scaled by setting the Scale field.
// This will scale the Delta and DeltaSecs values starting at the next frame.
type VirtualTime struct {
	Elapsed   time.Duration
	Delta     time.Duration
	DeltaSecs float64

	Scale float64
}

func updateVirtualTime(v *VirtualTime, lastTime *Local[time.Time]) {
	now := time.Now()

	if lastTime.Value.IsZero() {
		lastTime.Value = now
		return
	}

	delta := time.Duration(float64(now.Sub(lastTime.Value)) * v.Scale)
	lastTime.Value = now

	v.Delta = delta
	v.DeltaSecs = v.Delta.Seconds()
	v.Elapsed += v.Delta
}
