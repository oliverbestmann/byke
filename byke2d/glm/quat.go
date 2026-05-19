package glm

import "math"

type Quat struct {
	x, y, z, s1 float32
}

// QuatOf creates a Quat from its four components. This function does not check if the
// values are normalized, it is the responsibility of the user to provide normalized values
func QuatOf(a, b, c, d float32) Quat {
	return Quat{a, b, c, d - 1}
}

func (q Quat) Values() Vec4f {
	return Vec4f{q.x, q.y, q.z, q.s1 + 1}
}

// QuatFromAxisAngle creates a quaternion for a normalized rotation `axis` and `angle` (in radians).
// The axis must be a unit vector.
func QuatFromAxisAngle(axis Vec3f, angleRad Rad) Quat {
	// TODO validate

	sin, cos := math.Sincos(float64(angleRad) * -0.5)
	v := axis.Scale(float32(sin))
	return QuatOf(v[0], v[1], v[2], float32(cos))
}

// QuatFromScaledAxis creates a quaternion that rotates `v.Len()` radians around `v.Normalize()`.
// Results in the identity quaternion.
func QuatFromScaledAxis(v Vec3f) Quat {
	l := v.Length()
	if l == 0 {
		return Quat{}
	}

	return QuatFromAxisAngle(v.Scale(float32(1)/l), Rad(l))
}

func RotationXQuat(angle Rad) Quat {
	return QuatFromAxisAngle(Vec3f{1, 0, 0}, angle)
}

func RotationYQuat(angle Rad) Quat {
	return QuatFromAxisAngle(Vec3f{0, 1, 0}, angle)
}

func RotationZQuat(angle Rad) Quat {
	return QuatFromAxisAngle(Vec3f{0, 0, 1}, angle)
}

// QuatFromRotationArc gets the minimal rotation for transforming `from` to `to`.
// The rotation is in the plane spanned by the two vectors. Will rotate at most
// 180 degrees.
//
// The inputs must be unit vectors.
//
// `QuatFromRotationArc(from, to).Transform(from) ≈ to`.
//
// For near-singular cases (from≈to and from≈-to) the current implementation
// is only accurate to about 0.001 (for `f32`).
func QuatFromRotationArc(from, to Vec3f) Quat {
	const Epsilon = 1.19209290e-07
	const OneMinusEps = 1.0 - 2.0*Epsilon

	dot := from.Dot(to)
	if dot > OneMinusEps {
		// 0° singularity: from ≈ to
		return Quat{}
	} else if dot < -OneMinusEps {
		// 180° singularity: from ≈ -to
		return QuatFromAxisAngle(from.AnyOrthonormalVector(), math.Pi)
	}

	c := from.Cross(to)
	return QuatOf(Vec4f{c[0], c[1], c[2], 1.0 + dot}.Normalize().XYZW())
}

func (q Quat) Inverse() Quat {
	return Quat{
		-q.x,
		-q.y,
		-q.z,
		q.s1,
	}
}

// Dot calculates the dot product between the two quaternions.
// The dot product is equal to the cos of the angle between
// the two quaternions.
func (q Quat) Dot(other Quat) float32 {
	return q.ToVec4().Dot(other.ToVec4())
}

func (q Quat) ToVec4() Vec4f {
	return Vec4f{q.x, q.y, q.z, q.s1 + 1}
}

func (q Quat) AngleBetween(other Quat) Rad {
	return Rad(math.Acos(math.Abs(float64(q.Dot(other))))) * 2.0
}

func (q Quat) Mul(other Quat) Quat {
	qs := q.s1 + 1
	os := other.s1 + 1

	return QuatOf(
		qs*other.x+q.x*os+q.y*other.z-q.z*other.y,
		qs*other.y+q.y*os+q.z*other.x-q.x*other.z,
		qs*other.z+q.z*os+q.x*other.y-q.y*other.x,
		qs*os-q.x*other.x-q.y*other.y-q.z*other.z,
	)
}

func (q Quat) ToMat4() Mat4f {
	x, y, z, s := q.x, q.y, q.z, q.s1+1

	// Normalize quaternion
	n := float32(math.Sqrt(float64(x*x + y*y + z*z + s*s)))
	x /= n
	y /= n
	z /= n
	s /= n

	xx := x * x
	yy := y * y
	zz := z * z

	xs := x * s
	xy := x * y
	xz := x * z
	ys := y * s
	yz := y * z
	zs := z * s

	return Mat4f{
		{
			1 - 2*(yy+zz),
			2 * (xy - zs),
			2 * (xz + ys),
			0,
		},
		{
			2 * (xy + zs),
			1 - 2*(xx+zz),
			2 * (yz - xs),
			0,
		},
		{
			2 * (xz - ys),
			2 * (yz + xs),
			1 - 2*(xx+yy),
			0,
		},
		{
			0, 0, 0, 1,
		},
	}
}

func (q Quat) Transform(p Vec3f) Vec3f {
	return q.ToMat4().Transform3(p)
}
