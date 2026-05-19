//go:build nosimd || (!arm64 && !amd64)

package glm

func mat4fMulAssign(m, o *mat4f) {
	var mCopy = *m
	mat4fMulGo(&mCopy, o, m)
}

func mat4fScale(m *mat4f, x, y, z float32) {
	mat4fScaleGo(m, x, y, z)
}

func mat4fTranslate(m *mat4f, x, y, z float32) {
	mat4fTranslateGo(m, x, y, z)
}
