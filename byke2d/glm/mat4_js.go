//go:build js

package glm

func mat4fMulSimd(m, o, res *mat4f) {
	// for now just delegate to the go implementation
	mat4fMulGo(&m, &o, &res)
}
