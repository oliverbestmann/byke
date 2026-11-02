package glm

func (lhs Vec3f) Cross(rhs Vec3f) Vec3f {
	return Vec3f{
		lhs.Y*rhs.Z - rhs.Y*lhs.Z,
		lhs.Z*rhs.X - rhs.Z*lhs.X,
		lhs.X*rhs.Y - rhs.X*lhs.Y,
	}
}

func (lhs Vec3f) AnyOrthonormalVector() Vec3f {
	var sign float64
	switch {
	case lhs.Z > 0:
		sign = 1
	case lhs.Z < 0:
		sign = -1
	}

	a := -1.0 / (sign + float64(lhs.Z))
	b := float64(lhs.X*lhs.Y) * a

	return Vec3f{
		float32(b),
		float32(sign + float64(lhs.Y*lhs.Y)*a),
		-lhs.Z,
	}
}
