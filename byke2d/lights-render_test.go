package byke2d

import (
	"testing"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/stretchr/testify/require"
)

func TestDirectionalShadowProjection(t *testing.T) {
	// light at (0, 10, 0) shining straight down, box 4 wide, 6 high and 8 long
	view := lightWorldToView(glm.Vec3f{0, 10, 0}, glm.Vec3f{0, -1, 0})
	worldToClip := orthographicReverseZ(glm.Vec3f{4, 6, 8}).Mul(view)

	project := func(p glm.Vec3f) glm.Vec4f {
		return worldToClip.Transform(p.Extend(1))
	}

	// the light position is the center of the near plane
	center := project(glm.Vec3f{0, 10, 0})
	require.InDeltaSlice(t, []float32{0, 0, 1, 1}, center[:], 1e-5)

	// depth decreases along the light direction and reaches 0 at the far plane (reverse-z)
	require.InDelta(t, 0.5, project(glm.Vec3f{0, 6, 0})[2], 1e-5)
	require.InDelta(t, 0, project(glm.Vec3f{0, 2, 0})[2], 1e-5)

	// half the width and height map to the edges of clip space. For a light
	// shining straight down, view x is world z and view y is world x.
	corner := project(glm.Vec3f{3, 10, -2})
	require.InDelta(t, 1, abs32(corner[0]), 1e-5)
	require.InDelta(t, 1, abs32(corner[1]), 1e-5)
}

func abs32(v float32) float32 {
	return max(v, -v)
}
