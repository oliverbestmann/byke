package byke

import (
	"time"
)

type FixedTime struct {
	Elapsed   time.Duration
	Delta     time.Duration
	DeltaSecs float64

	StepInterval time.Duration

	overstep time.Duration
}

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
