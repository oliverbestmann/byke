package gm

import (
	"math"
	"math/rand/v2"
)

// RandomIn returns a random value uniformly sampled from the given range, excluding max.
func RandomIn[S Scalar](min, max S) S {
	return S(rand.Float64()*(float64(max)-float64(min))) + min
}

// RandomAngle returns a random angle uniformly sampled from the full circle
func RandomAngle() Rad {
	return Rad(RandomIn(0, 2*math.Pi))
}

// RandomVec returns a vector uniformly sampled from within the unit circle.
func RandomVec[S Scalar]() VecType[S] {
	for {
		v := VecType[S]{
			X: S(RandomIn(-1, 1)),
			Y: S(RandomIn(-1, 1)),
		}

		if v.LengthSqr() <= 1 {
			return v
		}
	}
}
