package glm

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func runTest(t *testing.T, op func(r *rand.Rand, m mat4f) (expected mat4f, actual mat4f)) {
	r := rand.New(rand.NewPCG(1, 1))

	for range 250_000 {
		m := randMat4f(r)

		expected, actual := op(r, m)

		for i := range 4 {
			require.InDelta(t, expected[i][0], actual[i][0], 1e-5)
			require.InDelta(t, expected[i][1], actual[i][1], 1e-5)
			require.InDelta(t, expected[i][2], actual[i][2], 1e-5)
			require.InDelta(t, expected[i][3], actual[i][3], 1e-5)
		}
	}
}

func randMat4f(r *rand.Rand) mat4f {
	return mat4f{
		{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
		{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
		{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
		{r.Float32(), r.Float32(), r.Float32(), r.Float32()},
	}
}

func TestMulSimd(t *testing.T) {
	runTest(t, func(r *rand.Rand, m mat4f) (expected mat4f, actual mat4f) {
		other := randMat4f(r)

		mat4fMulGo(&m, &other, &expected)

		actual = m
		mat4fMulAssign(&actual, &other)

		return
	})
}

func TestMatScale(t *testing.T) {
	runTest(t, func(r *rand.Rand, m mat4f) (expected mat4f, actual mat4f) {
		x, y, z := r.Float32(), r.Float32(), r.Float32()

		expected = m
		mat4fScaleGo(&expected, x, y, z)

		actual = m
		mat4fScale(&actual, x, y, z)

		return
	})
}

func TestMatTranslate(t *testing.T) {
	runTest(t, func(r *rand.Rand, m mat4f) (expected mat4f, actual mat4f) {
		x, y, z := r.Float32(), r.Float32(), r.Float32()

		expected = m
		mat4fTranslateGo(&expected, x, y, z)

		actual = m
		mat4fTranslate(&actual, x, y, z)

		return
	})
}

func BenchmarkMultiplyMat4(b *testing.B) {
	m := ScaleMat4f(1, 2, 3)

	b.ReportAllocs()

	for range b.N {
		_ = m.Mul(m)
	}
}

func Benchmark_Mat4fMulGo(b *testing.B) {
	m := randMat4f(rand.New(rand.NewPCG(1, 1)))

	b.ReportAllocs()

	for range b.N {
		var out mat4f
		mat4fMulGo(&m, &m, &out)
	}
}

func Benchmark_Mat4fMulAssign(b *testing.B) {
	m := randMat4f(rand.New(rand.NewPCG(1, 1)))

	b.ReportAllocs()

	for range b.N {
		mCopy := m
		mat4fMulAssign(&mCopy, &mCopy)
	}
}

func Benchmark_Mat4fScale(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 1))

	m := randMat4f(r)
	x, y, z := r.Float32(), r.Float32(), r.Float32()

	b.ReportAllocs()

	for range b.N {
		mCopy := m
		mat4fScale(&mCopy, x, y, z)
	}
}

func Benchmark_Mat4fScaleGo(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 1))

	m := randMat4f(r)
	x, y, z := r.Float32(), r.Float32(), r.Float32()

	b.ReportAllocs()

	for range b.N {
		mCopy := m
		mat4fScaleGo(&mCopy, x, y, z)
	}
}

func Benchmark_Mat4fTranslate(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 1))

	m := randMat4f(r)
	x, y, z := r.Float32(), r.Float32(), r.Float32()

	b.ReportAllocs()

	for range b.N {
		mCopy := m
		mat4fTranslate(&mCopy, x, y, z)
	}
}

func Benchmark_Mat4fTranslateGo(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 1))

	m := randMat4f(r)
	x, y, z := r.Float32(), r.Float32(), r.Float32()

	b.ReportAllocs()

	for range b.N {
		mCopy := m
		mat4fTranslateGo(&mCopy, x, y, z)
	}
}
