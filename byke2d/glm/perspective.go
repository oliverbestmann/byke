package glm

import "math"

// Perspective returns an infinite reverse-z perspective projection: near maps to
// depth 1 and infinity to depth 0. far is ignored.
func Perspective(fovY Rad, aspect, near, far float32) Mat4f {
	f := float32(1 / math.Tan(float64(fovY*0.5)))

	return Mat4f{
		{f / aspect, 0, 0, 0},
		{0, f, 0, 0},
		{0, 0, 0, -1},
		{0, 0, near, 0},
	}
}

func LookAt(eye, center, up Vec3f) Mat4f {
	f := center.Sub(eye).Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)

	return Mat4f{
		{s.X, u.X, -f.X, 0},
		{s.Y, u.Y, -f.Y, 0},
		{s.Z, u.Z, -f.Z, 0},
		{-eye.Dot(s), -eye.Dot(u), eye.Dot(f), 1},
	}
}

func DegToRad[T Numeric](deg T) Rad {
	return Rad(float64(deg) * (math.Pi / 180))
}

func RadToDeg[T Numeric](rad Rad) (deg T) {
	return T(rad * (180 / math.Pi))
}
