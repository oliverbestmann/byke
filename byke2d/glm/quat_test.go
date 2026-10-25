package glm

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuatRotate(t *testing.T) {
	q := RotationZQuat(math.Pi / 2)

	res := q.Transform(Vec3f{1, 0, 0})
	require.InDelta(t, 0, res[0], 1e-5)
	require.InDelta(t, 1, res[1], 1e-5)
	require.InDelta(t, 0, res[2], 1e-5)

	res = q.Transform(Vec3f{0, 1, 0})
	require.InDelta(t, -1, res[0], 1e-5)
	require.InDelta(t, 0, res[1], 1e-5)
	require.InDelta(t, 0, res[2], 1e-5)
}

func TestQuatFromRotationArc(t *testing.T) {
	pairs := [][2]Vec3f{
		{{1, 0, 0}, {0, 1, 0}},
		{{0, 0, 1}, Vec3f{1, 2, 3}.Normalize()},
		{Vec3f{1, -1, 0}.Normalize(), Vec3f{0, 1, 1}.Normalize()},
	}

	for _, p := range pairs {
		got := QuatFromRotationArc(p[0], p[1]).Transform(p[0])
		require.Less(t, got.Sub(p[1]).Length(), float32(1e-5))
	}
}

func TestQuatMulMatchesMat4Mul(t *testing.T) {
	a := RotationXQuat(0.7)
	b := RotationYQuat(1.3)

	got := a.Mul(b).ToMat4()
	expected := a.ToMat4().Mul(b.ToMat4())

	for c := range 4 {
		for r := range 4 {
			require.InDelta(t, expected[c][r], got[c][r], 1e-5)
		}
	}
}

func TestQuatValuesStandardLayout(t *testing.T) {
	// 90° around z: (0, 0, sin(45°), cos(45°))
	v := RotationZQuat(math.Pi / 2).Values()
	require.InDelta(t, 0, v[0], 1e-6)
	require.InDelta(t, 0, v[1], 1e-6)
	require.InDelta(t, math.Sqrt2/2, v[2], 1e-6)
	require.InDelta(t, math.Sqrt2/2, v[3], 1e-6)
}

func TestQuatFromMat3RoundTrip(t *testing.T) {
	qs := []Quat{
		IdentityQuat(),
		RotationXQuat(0.7),
		RotationYQuat(2.5),
		RotationZQuat(-2.9),
		RotationXQuat(1.1).Mul(RotationYQuat(2.2)).Mul(RotationZQuat(3.0)),
		QuatFromAxisAngle(Vec3f{1, 1, 1}.Normalize(), 3.1),
	}

	for _, q := range qs {
		m4 := q.ToMat4()

		var m Mat3f
		for i := range 3 {
			for j := range 3 {
				m[i][j] = m4[i][j]
			}
		}

		got := QuatFromMat3(m)

		require.InDelta(t, 1.0, math.Abs(float64(got.Dot(q))), 0.001)

		for _, p := range []Vec3f{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 2, 3}} {
			a, b := q.Transform(p), got.Transform(p)
			require.Less(t, a.Sub(b).Length(), float32(0.001))
		}
	}
}
