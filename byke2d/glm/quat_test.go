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
