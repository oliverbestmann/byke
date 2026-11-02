package glm

import "math"

type mat4f = [4][4]float32

//goland:noinspection DuplicatedCode
func mat4fMulGo(m, o, res *mat4f) {
	res[0][0] = m[0][0]*o[0][0] + m[1][0]*o[0][1] + m[2][0]*o[0][2] + m[3][0]*o[0][3]
	res[0][1] = m[0][1]*o[0][0] + m[1][1]*o[0][1] + m[2][1]*o[0][2] + m[3][1]*o[0][3]
	res[0][2] = m[0][2]*o[0][0] + m[1][2]*o[0][1] + m[2][2]*o[0][2] + m[3][2]*o[0][3]
	res[0][3] = m[0][3]*o[0][0] + m[1][3]*o[0][1] + m[2][3]*o[0][2] + m[3][3]*o[0][3]

	res[1][0] = m[0][0]*o[1][0] + m[1][0]*o[1][1] + m[2][0]*o[1][2] + m[3][0]*o[1][3]
	res[1][1] = m[0][1]*o[1][0] + m[1][1]*o[1][1] + m[2][1]*o[1][2] + m[3][1]*o[1][3]
	res[1][2] = m[0][2]*o[1][0] + m[1][2]*o[1][1] + m[2][2]*o[1][2] + m[3][2]*o[1][3]
	res[1][3] = m[0][3]*o[1][0] + m[1][3]*o[1][1] + m[2][3]*o[1][2] + m[3][3]*o[1][3]

	res[2][0] = m[0][0]*o[2][0] + m[1][0]*o[2][1] + m[2][0]*o[2][2] + m[3][0]*o[2][3]
	res[2][1] = m[0][1]*o[2][0] + m[1][1]*o[2][1] + m[2][1]*o[2][2] + m[3][1]*o[2][3]
	res[2][2] = m[0][2]*o[2][0] + m[1][2]*o[2][1] + m[2][2]*o[2][2] + m[3][2]*o[2][3]
	res[2][3] = m[0][3]*o[2][0] + m[1][3]*o[2][1] + m[2][3]*o[2][2] + m[3][3]*o[2][3]

	res[3][0] = m[0][0]*o[3][0] + m[1][0]*o[3][1] + m[2][0]*o[3][2] + m[3][0]*o[3][3]
	res[3][1] = m[0][1]*o[3][0] + m[1][1]*o[3][1] + m[2][1]*o[3][2] + m[3][1]*o[3][3]
	res[3][2] = m[0][2]*o[3][0] + m[1][2]*o[3][1] + m[2][2]*o[3][2] + m[3][2]*o[3][3]
	res[3][3] = m[0][3]*o[3][0] + m[1][3]*o[3][1] + m[2][3]*o[3][2] + m[3][3]*o[3][3]
}

// mat4fMulAssignGo multiplies m by o and stores the result in m. m and o may alias.
func mat4fMulAssignGo(m, o *mat4f) {
	mCopy, oCopy := *m, *o
	mat4fMulGo(&mCopy, &oCopy, m)
}

//goland:noinspection DuplicatedCode
func mat4fTranslateGo(m *mat4f, x, y, z float32) {
	m[3][0] = m[3][0] + m[0][0]*x + m[1][0]*y + m[2][0]*z
	m[3][1] = m[3][1] + m[0][1]*x + m[1][1]*y + m[2][1]*z
	m[3][2] = m[3][2] + m[0][2]*x + m[1][2]*y + m[2][2]*z
	m[3][3] = m[3][3] + m[0][3]*x + m[1][3]*y + m[2][3]*z
}

//goland:noinspection DuplicatedCode
func mat4fScaleGo(m *mat4f, x, y, z float32) {
	m[0][0] = m[0][0] * x
	m[0][1] = m[0][1] * x
	m[0][2] = m[0][2] * x
	m[0][3] = m[0][3] * x

	m[1][0] = m[1][0] * y
	m[1][1] = m[1][1] * y
	m[1][2] = m[1][2] * y
	m[1][3] = m[1][3] * y

	m[2][0] = m[2][0] * z
	m[2][1] = m[2][1] * z
	m[2][2] = m[2][2] * z
	m[2][3] = m[2][3] * z
}

func (m Mat4f) Translation() Vec3f {
	return Vec3f{
		m[3][0],
		m[3][1],
		m[3][2],
	}
}

// TransformBox transforms the axis aligned box given by its center and half size by m.
// It returns the axis aligned box that encloses the result, which is larger than the
// box itself if m rotates.
func (m *Mat4f) TransformBox(center, half Vec3f) (Vec3f, Vec3f) {
	return mat4fTransformBox((*mat4f)(m), center, half)
}

func absf32(v float32) float32 {
	// compiles to a single FABS/ANDPS, unlike the float32 bit mask
	return float32(math.Abs(float64(v)))
}

//goland:noinspection DuplicatedCode
func mat4fTransformBoxGo(m *mat4f, c, h Vec3f) (Vec3f, Vec3f) {
	center := Vec3f{
		m[0][0]*c.X + m[1][0]*c.Y + m[2][0]*c.Z + m[3][0],
		m[0][1]*c.X + m[1][1]*c.Y + m[2][1]*c.Z + m[3][1],
		m[0][2]*c.X + m[1][2]*c.Y + m[2][2]*c.Z + m[3][2],
	}

	half := Vec3f{
		absf32(m[0][0])*h.X + absf32(m[1][0])*h.Y + absf32(m[2][0])*h.Z,
		absf32(m[0][1])*h.X + absf32(m[1][1])*h.Y + absf32(m[2][1])*h.Z,
		absf32(m[0][2])*h.X + absf32(m[1][2])*h.Y + absf32(m[2][2])*h.Z,
	}

	return center, half
}
