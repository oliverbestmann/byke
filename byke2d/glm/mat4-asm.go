//go:build !nosimd && ((amd64 && !goexperiment.simd) || arm64)

package glm

// mat4fMul is defined in the corresponding assembly file
//
//go:noescape
func mat4fMulAssign(m, o *mat4f)

//go:noescape
func mat4fScaleAssign(m *mat4f, x, y, z float32)

//go:noescape
func mat4fTranslateAssign(m *mat4f, x, y, z float32)
