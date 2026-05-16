package glm

func (lhs Vec3f) Cross(rhs Vec3f) Vec3f {
	return Vec3f{
		lhs[1]*rhs[2] - rhs[1]*lhs[2],
		lhs[2]*rhs[0] - rhs[2]*lhs[0],
		lhs[0]*rhs[1] - rhs[0]*lhs[1],
	}
}

func (lhs Vec3f) AnyOrthonormalVector() Vec3f {
	var sign float64
	switch {
	case lhs[2] > 0:
		sign = 1
	case lhs[2] < 0:
		sign = -1
	}

	a := -1.0 / (sign + float64(lhs[2]))
	b := float64(lhs[0]*lhs[1]) * a

	return Vec3f{
		float32(b),
		float32(sign + float64(lhs[1]*lhs[1])*a),
		-lhs[2],
	}
}
