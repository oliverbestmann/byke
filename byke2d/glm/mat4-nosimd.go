//go:build nosimd || (!arm64 && !amd64)

package glm

func mat4fMulAssign(m, o *mat4f) {
	var mCopy = *m
	mat4fMulGo(&mCopy, o, m)
}

func mat4fScaleAssign(m *mat4f, x, y, z float32) {
	mat4fScaleAssignGo(m, x, y, z)
}

func mat4fTranslateAssign(m *mat4f, x, y, z float32) {
	mat4fTranslateAssignGo(m, x, y, z)
}
