//go:build (amd64 && !goexperiment.simd) || arm64

package glm

// mat4fMulSimd is defined in the corresponding assembly file
//
//go:noescape
func mat4fMulSimd(m, o, res *mat4f)
