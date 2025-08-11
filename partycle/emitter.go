package partycle

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
)

var particleMesh = bykebiten.Rectangle(gm.VecOne)

func emitterSystem(
	vt byke.VirtualTime,
	commands *byke.Commands,
	emitters byke.Query[struct {
		Emitter   *Emitter
		Transform bykebiten.GlobalTransform
		Layer     byke.Option[bykebiten.Layer]
	}],
) {
	for item := range emitters.Items() {
		e := item.Emitter

		if !e.previousInitialized {
			e.previous = item.Transform.Translation
			e.previousInitialized = true
		}

		previousTranslation := e.previous
		e.previous = item.Transform.Translation

		if e.Disabled {
			continue
		}

		item.Emitter.spawnAcc += jitterValue(e.ParticlesPerSecond, e.ParticlesPerSecondJitter) * vt.Scale

		var visuals byke.ErasedComponent = &particleMesh

		for item.Emitter.spawnAcc > 1 {
			item.Emitter.spawnAcc -= 1

			lifetime := jitterValue(e.ParticleLifetime, e.ParticleLifetimeJitter)
			if lifetime <= 0 {
				continue
			}

			particle := Particle{
				Lifetime:            byke.NewTimer(lifetime, byke.TimerModeOnce),
				LinearVelocity:      jitterVec(e.LinearVelocity, e.LinearVelocityJitter),
				LinearAcceleration:  jitterVec(e.LinearAcceleration, e.LinearAccelerationJitter),
				LinearDampening:     jitterValue(e.DampeningLinear, e.DampeningLinearJitter),
				AngularVelocity:     jitterValue(e.AngularVelocity, e.AngularVelocityJitter),
				AngularAcceleration: jitterValue(e.AngularAcceleration, e.AngularAccelerationJitter),
				AngularDampening:    jitterValue(e.DampeningAngular, e.DampeningAngularJitter),
			}

			if e.ColorCurve.HasValues() {
				particle.ColorCurve = e.ColorCurve
			} else {
				particle.ColorCurve = StaticValueCurve(color.White)
			}

			if e.ScaleCurve.HasValues() {
				particle.ScaleCurve = e.ScaleCurve
			} else {
				particle.ScaleCurve = StaticValueCurve(gm.VecOne)
			}

			if e.Visual != nil {
				visuals = e.Visual()
			}

			// interpolate along the path moved between the frame
			pos := item.Transform.Translation.Sub(previousTranslation).
				Mul(gm.RandomIn(0.0, 1.0)).
				Add(previousTranslation).
				Add(gm.RandomVec[float64]().Mul(item.Emitter.Radius))

			rot := jitterValue(e.Rotation, e.RotationJitter)

			transform := bykebiten.TransformFromXY(pos.XY()).
				WithRotation(rot).
				WithScale(particle.ScaleCurve.ValueAt(0).XY())

			commands.Spawn(
				transform,
				visuals,
				particle,
				item.Layer.OrZero(),
			)
		}
	}
}

func StaticValueCurve[T any](value T) Curve[T] {
	return Curve[T]{
		Values: []CurveValue[T]{
			{
				Value: value,
			},
		},
	}
}

func EquidistantCurve[T any](lerper Lerper[T], firstValue, secondValue T, values ...T) Curve[T] {
	divider := float64(len(values) + 2 - 1)

	curveValues := make([]CurveValue[T], 0, len(values)+2)

	curveValues = append(curveValues,
		CurveValue[T]{
			Time:  float64(0) / divider,
			Value: firstValue,
		},
		CurveValue[T]{
			Time:  float64(1) / divider,
			Value: secondValue,
		},
	)

	for idx, value := range values {
		curveValues = append(curveValues, CurveValue[T]{
			Time:  float64(idx+2) / divider,
			Value: value,
		})
	}

	return Curve[T]{
		Lerper: lerper,
		Values: curveValues,
	}
}

func jitterValue[T gm.Scalar](base, jitter T) T {
	return base + gm.RandomIn(-jitter, jitter)
}

func jitterVec(base, jitter gm.Vec) gm.Vec {
	return base.Add(gm.RandomVec[float64]().MulEach(jitter))
}
