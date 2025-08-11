package partycle

import (
	"time"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
)

type Particle struct {
	byke.Component[Particle]

	Lifetime byke.Timer

	LinearVelocity  gm.Vec
	AngularVelocity gm.Rad

	LinearAcceleration  gm.Vec
	AngularAcceleration gm.Rad

	LinearDampening  float64
	AngularDampening float64

	ColorCurve Curve[color.Color]
	ScaleCurve Curve[gm.Vec]
}

type Emitter struct {
	byke.Component[Emitter]

	ParticlesPerSecond       float64
	ParticlesPerSecondJitter float64

	LinearVelocity       gm.Vec
	LinearVelocityJitter gm.Vec

	LinearAcceleration       gm.Vec
	LinearAccelerationJitter gm.Vec

	AngularVelocity       gm.Rad
	AngularVelocityJitter gm.Rad

	AngularAcceleration       gm.Rad
	AngularAccelerationJitter gm.Rad

	Rotation       gm.Rad
	RotationJitter gm.Rad

	DampeningLinear        float64
	DampeningLinearJitter  float64
	DampeningAngular       float64
	DampeningAngularJitter float64

	ParticleLifetime       time.Duration
	ParticleLifetimeJitter time.Duration

	// defaults to constant white
	ColorCurve Curve[color.Color]

	// defaults to constant 1
	ScaleCurve Curve[gm.Vec]

	// Positive radius makes the Emitter a circle
	Radius float64

	// Set Visual to a function providing a visual for the particle. An implementation
	// might return a Sprite, a Mesh, a Shader, etc. To return multiple components,
	// use byke.BundleOf
	// If not set, a 1x1 white rectangle mesh will be used.
	Visual func() byke.ErasedComponent

	// accumulator for number of particles to spawn
	spawnAcc float64

	// the previous position in the last frame.
	// we use this to emit particles somewhere "between" the frames
	previous            gm.Vec
	previousInitialized bool

	Disabled bool
}

type Curve[T any] struct {
	// If the Lerper is nil, no lerping will be performed and the
	// nearest Value will be used. This is especially fine for
	// static "one value" curves
	Lerper Lerper[T]

	Values []CurveValue[T]
}

func (c Curve[T]) HasValues() bool {
	return len(c.Values) > 0
}

func (c Curve[T]) ValueAt(t float64) T {
	if len(c.Values) == 0 {
		var zeroValue T
		return zeroValue
	}

	if len(c.Values) == 1 {
		return c.Values[0].Value
	}

	for idx := 0; idx < len(c.Values)-1; idx++ {
		lhs := c.Values[idx]
		rhs := c.Values[idx+1]
		if t >= rhs.Time {
			continue
		}

		if c.Lerper == nil {
			return lhs.Value
		}

		f := (t - lhs.Time) / (rhs.Time - lhs.Time)
		return c.Lerper(f, lhs.Value, rhs.Value)
	}

	// use the value before the first time
	if t < c.Values[0].Time {
		return c.Values[0].Value
	}

	// time is above the last value
	lastValue := c.Values[len(c.Values)-1]
	if t > lastValue.Time {
		return lastValue.Value
	}

	// probably invalid curve, return any configured value
	return c.Values[0].Value
}

type CurveValue[T any] struct {
	Time  float64
	Value T
}
