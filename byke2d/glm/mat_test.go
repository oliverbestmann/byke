package glm

import (
	"testing"
)

func BenchmarkMultiplyMat4(b *testing.B) {
	m := ScaleMat4f(1, 2, 3)

	b.ReportAllocs()

	for range b.N {
		_ = m.Mul(m)
	}
}

func BenchmarkMultiplyMat4New(b *testing.B) {
	m := mat4Scale(1, 2, 3)

	b.ReportAllocs()

	for range b.N {
		_ = m.Mul(m)
	}
}

func BenchmarkMultiplyMat4NewSimd(b *testing.B) {
	m := mat4Scale(1, 2, 3)

	b.ReportAllocs()

	for range b.N {
		_ = m.MulSimd(m)
	}
}
