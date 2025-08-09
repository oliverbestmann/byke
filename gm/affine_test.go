package gm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffine_Transform(t *testing.T) {
	tr := IdentityAffine().Translate(Vec{X: 2.0, Y: 1.0})
	require.Equal(t, Vec{X: 12.0, Y: 11.0}, tr.Transform(Vec{X: 10.0, Y: 10.0}))

	// translate vector by (10, 0) first, then rotate by 90°
	tr = IdentityAffine().Translate(Vec{X: 10.0, Y: 0.0}).Rotate(DegToRad(90))
	require.Equal(t, Vec{X: 10, Y: 1}, tr.Transform(Vec{X: 1, Y: 0}))

	// rotate by 90° first, then move by (in local space) (10, 0)
	tr = IdentityAffine().Rotate(DegToRad(90)).Translate(Vec{X: 10.0, Y: 0.0})
	res := tr.Transform(Vec{X: 1, Y: 0})
	require.InDelta(t, 0.0, res.X, 1e-9)
	require.InDelta(t, 11.0, res.Y, 1e-9)

	// scale by 2 first, then move by local 5 (10 real)
	tr = IdentityAffine().Scale(VecSplat(2.0)).Translate(Vec{X: 5})
	res = tr.Transform(Vec{X: 10})
	require.InDelta(t, 30.0, res.X, 1e-9)
}
