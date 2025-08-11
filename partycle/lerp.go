package partycle

import (
	"github.com/oliverbestmann/byke/bykebiten/color"
	"github.com/oliverbestmann/byke/gm"
)

// Lerper does a linear interpolation between lhs and rhs using
// the factor f. A value for f of 0 returns lhs, a value of 1 returns rhs.
//
// Use an easing function to calculate f to perform
// custom interpolations between the values
type Lerper[T any] func(f float64, rhs, lhs T) T

func LerpVec(f float64, lhs, rhs gm.Vec) gm.Vec {
	return lhs.Add(rhs.Sub(lhs).Mul(f))
}

func LerpFloat[T ~float32 | ~float64](f float64, lhs, rhs T) T {
	return (rhs-lhs)*T(f) + lhs
}

func LerpColor(f float64, lhs, rhs color.Color) color.Color {
	return color.Color{
		R: LerpFloat(f, lhs.R, rhs.R),
		G: LerpFloat(f, lhs.G, rhs.G),
		B: LerpFloat(f, lhs.B, rhs.B),
		A: LerpFloat(f, lhs.A, rhs.A),
	}
}

func LerpAngle(f float64, lhs, rhs gm.Rad) gm.Rad {
	d := lhs.DifferenceTo(rhs)
	return lhs + gm.Rad(f)*d
}
