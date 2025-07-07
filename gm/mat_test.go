package gm

import (
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestMat_Inverse(t *testing.T) {
	m := RotationMat(2)
	require.NotEqual(t, m, m.Inverse())
	require.Equal(t, m, m.Inverse().Inverse())
}

func TestMat_InverseIdentity(t *testing.T) {
	m := IdentityMat()
	require.Equal(t, m, m.Inverse())
}

func TestMat_Mul(t *testing.T) {
	m := RotationMat(math.Pi).Mul(RotationMat(math.Pi / 2))
	require.Equal(t, m, RotationMat(math.Pi*1.5))
}

func TestMat_Transform(t *testing.T) {
	t.Run("rotate 180°", func(t *testing.T) {
		m := RotationMat(math.Pi)

		r := m.Transform(Vec{X: 1, Y: 1})
		require.InDelta(t, -1, r.X, 1e-6)
		require.InDelta(t, -1, r.Y, 1e-6)

		r = m.Transform(Vec{X: 0, Y: 1})
		require.InDelta(t, 0, r.X, 1e-6)
		require.InDelta(t, -1, r.Y, 1e-6)
	})

	t.Run("rotate 90°", func(t *testing.T) {
		m := RotationMat(math.Pi / 2)

		r := m.Transform(Vec{X: 1, Y: 1})
		require.InDelta(t, -1, r.X, 1e-6)
		require.InDelta(t, 1, r.Y, 1e-6)

		r = m.Transform(Vec{X: 1, Y: 0})
		require.InDelta(t, 0, r.X, 1e-6)
		require.InDelta(t, 1, r.Y, 1e-6)

		r = m.Transform(Vec{X: 0, Y: 1})
		require.InDelta(t, -1, r.X, 1e-6)
		require.InDelta(t, 0, r.Y, 1e-6)
	})
}
