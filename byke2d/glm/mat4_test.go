package glm

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMulSimd(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 1))

	for range 250_000 {
		m := mat4f{
			vec4f{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
			vec4f{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
			vec4f{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
			vec4f{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
		}

		var expected, res mat4f
		mat4fMulGo(&m, &m, &expected)
		mat4fMulSimd(&m, &m, &res)

		for i := range 4 {
			require.InEpsilon(t, expected[i].X, res[i].X, 1e-5)
			require.InEpsilon(t, expected[i].Y, res[i].Y, 1e-5)
			require.InEpsilon(t, expected[i].Z, res[i].Z, 1e-5)
			require.InEpsilon(t, expected[i].W, res[i].W, 1e-5)
		}
	}
}

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
