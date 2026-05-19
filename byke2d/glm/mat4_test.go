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
			{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
			{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
			{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
			{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
		}

		var expected, res mat4f
		mat4fMulGo(&m, &m, &expected)
		mat4fMul(&m, &m, &res)

		for i := range 4 {
			require.InEpsilon(t, expected[i][0], res[i][0], 1e-5)
			require.InEpsilon(t, expected[i][1], res[i][1], 1e-5)
			require.InEpsilon(t, expected[i][2], res[i][2], 1e-5)
			require.InEpsilon(t, expected[i][3], res[i][3], 1e-5)
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
		var out mat4f
		mat4fMulGo(&m, &m, &out)
	}
}

func BenchmarkMultiplyMat4NewSimd(b *testing.B) {
	m := mat4Scale(1, 2, 3)

	b.ReportAllocs()

	for range b.N {
		var out mat4f
		mat4fMul(&m, &m, &out)
	}
}

func mat4Scale(x, y, z float32) mat4f {
	var m mat4f
	m[0][0] = x
	m[1][1] = y
	m[2][2] = z
	m[3][3] = 1
	return m
}
