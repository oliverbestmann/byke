//go:build !nosimd && (amd64 || arm64 || (wasm && goexperiment.simd))

package glm

// The *Simd functions come from mat4-asm.go or mat4-simd.go. They need
// CPU features that are checked at runtime via useSimd.

func mat4fMulAssign(m, o *mat4f) {
	if !useSimd {
		mat4fMulAssignGo(m, o)
		return
	}

	mat4fMulAssignSimd(m, o)
}

func mat4fScale(m *mat4f, x, y, z float32) {
	if !useSimd {
		mat4fScaleGo(m, x, y, z)
		return
	}

	mat4fScaleSimd(m, x, y, z)
}

func mat4fTranslate(m *mat4f, x, y, z float32) {
	if !useSimd {
		mat4fTranslateGo(m, x, y, z)
		return
	}

	mat4fTranslateSimd(m, x, y, z)
}
