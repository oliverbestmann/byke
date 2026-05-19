package glm

import "math"

func Perspective[T float](fovY Rad, aspect, near, far float32) Mat4f {
	f := float32(1 / math.Tan(float64(fovY*0.5)))

	return Mat4f{
		{f / aspect, 0, 0, 0},
		{0, f, 0, 0},
		{0, 0, (far + near) / (near - far), -1},
		{0, 0, (2 * far * near) / (near - far), 0},
	}
}

func LookAt(eye, center, up Vec3f) Mat4f {
	f := (center.Sub(eye)).Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)

	return Mat4f{
		{s[0], u[0], -f[0], 0},
		{s[1], u[1], -f[1], 0},
		{s[2], u[2], -f[2], 0},
		{-eye.Dot(s), -eye.Dot(u), eye.Dot(f), 1},
	}
}

func DegToRad[T Numeric](deg T) Rad {
	return Rad(float64(deg) * (math.Pi / 180))
}

func RadToDeg[T Numeric](rad Rad) (deg T) {
	return T(rad * (180 / math.Pi))
}
