//go:build nosimd || !(amd64 || arm64 || (wasm && goexperiment.simd))

package glm

func mat4fMulAssign(m, o *mat4f) {
	mat4fMulAssignGo(m, o)
}

func mat4fScale(m *mat4f, x, y, z float32) {
	mat4fScaleGo(m, x, y, z)
}

func mat4fTranslate(m *mat4f, x, y, z float32) {
	mat4fTranslateGo(m, x, y, z)
}
