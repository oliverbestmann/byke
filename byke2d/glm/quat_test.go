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
