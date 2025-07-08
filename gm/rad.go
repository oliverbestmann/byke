package gm

import "math"

type Rad float64

func (r Rad) Degrees() float64 {
	return float64(r) * (180 / math.Pi)
}

// Radians returns the value of the angle in radians as float64.
func (r Rad) Radians() float64 {
	return float64(r)
}

// Normalized returns the angle normalized to the range [-π, π)
func (r Rad) Normalized() Rad {
	angle := float64(r)

	angle = math.Mod(angle+math.Pi, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	return Rad(angle - math.Pi)
}

// DifferenceTo returns the smallest difference between to angles
// normalized to the range [-π, π)
func (r Rad) DifferenceTo(other Rad) Rad {
	return (r - other).Normalized()
}

// Cos returns the cosine of the angle.
func (r Rad) Cos() float64 {
	return math.Cos(float64(r))
}

// Sin returns the sine of the angle.
func (r Rad) Sin() float64 {
	return math.Sin(float64(r))
}

func DegToRad(deg float64) Rad {
	return Rad(math.Pi / 180 * deg)
}
