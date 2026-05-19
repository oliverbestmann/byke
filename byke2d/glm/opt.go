package glm

func (m *Mat4f) TranslateAssign(x, y, z float32) {
	mat4fTranslate(
		(*mat4f)(m),
		x, y, z,
	)
}

func (m *Mat4f) ScaleAssign(x, y, z float32) {
	mat4fScale(
		(*mat4f)(m),
		x, y, z,
	)
}

func (m *Mat4f) RotateAssign(q Quat) {
	qm := q.ToMat4()
	mat4fMulAssign(
		(*mat4f)(m),
		(*mat4f)(&qm),
	)
}
