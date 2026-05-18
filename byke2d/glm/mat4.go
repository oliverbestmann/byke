package glm

type vec4f struct {
	X, Y, Z, W float32
}

func (a *vec4f) Dot(b *vec4f) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}

type mat4f [4]vec4f

func (m mat4f) Column(idx int) vec4f {
	return m[idx]
}

func (m mat4f) Mul(o mat4f) mat4f {
	var res mat4f
	mat4fMulGo(&m, &o, &res)
	return res
}

func (m mat4f) MulSimd(o mat4f) mat4f {
	var res mat4f
	mat4fMulSimd(&m, &o, &res)
	return res
}

func mat4Scale(x, y, z float32) mat4f {
	var m mat4f
	m[0].X = x
	m[1].Y = y
	m[2].Z = z
	m[3].W = 1
	return m
}

// Pure go implementation of matrix multiplication. We use this to check
// the simd implementation
//
//goland:noinspection DuplicatedCode
func mat4fMulGo(m, o, res *mat4f) {
	res[0].X = m[0].X*o[0].X + m[1].X*o[0].Y + m[2].X*o[0].Z + m[3].X*o[0].W
	res[0].Y = m[0].Y*o[0].X + m[1].Y*o[0].Y + m[2].Y*o[0].Z + m[3].Y*o[0].W
	res[0].Z = m[0].Z*o[0].X + m[1].Z*o[0].Y + m[2].Z*o[0].Z + m[3].Z*o[0].W
	res[0].W = m[0].W*o[0].X + m[1].W*o[0].Y + m[2].W*o[0].Z + m[3].W*o[0].W

	res[1].X = m[0].X*o[1].X + m[1].X*o[1].Y + m[2].X*o[1].Z + m[3].X*o[1].W
	res[1].Y = m[0].Y*o[1].X + m[1].Y*o[1].Y + m[2].Y*o[1].Z + m[3].Y*o[1].W
	res[1].Z = m[0].Z*o[1].X + m[1].Z*o[1].Y + m[2].Z*o[1].Z + m[3].Z*o[1].W
	res[1].W = m[0].W*o[1].X + m[1].W*o[1].Y + m[2].W*o[1].Z + m[3].W*o[1].W

	res[2].X = m[0].X*o[2].X + m[1].X*o[2].Y + m[2].X*o[2].Z + m[3].X*o[2].W
	res[2].Y = m[0].Y*o[2].X + m[1].Y*o[2].Y + m[2].Y*o[2].Z + m[3].Y*o[2].W
	res[2].Z = m[0].Z*o[2].X + m[1].Z*o[2].Y + m[2].Z*o[2].Z + m[3].Z*o[2].W
	res[2].W = m[0].W*o[2].X + m[1].W*o[2].Y + m[2].W*o[2].Z + m[3].W*o[2].W

	res[3].X = m[0].X*o[3].X + m[1].X*o[3].Y + m[2].X*o[3].Z + m[3].X*o[3].W
	res[3].Y = m[0].Y*o[3].X + m[1].Y*o[3].Y + m[2].Y*o[3].Z + m[3].Y*o[3].W
	res[3].Z = m[0].Z*o[3].X + m[1].Z*o[3].Y + m[2].Z*o[3].Z + m[3].Z*o[3].W
	res[3].W = m[0].W*o[3].X + m[1].W*o[3].Y + m[2].W*o[3].Z + m[3].W*o[3].W
}
