package byke2d

import (
	"testing"

	"github.com/oliverbestmann/pulse/glm"
	"github.com/stretchr/testify/require"
)

func TestGlobalTransform_Affine2(t *testing.T) {
	var pos glm.Vec3f

	tr := GlobalTransform{
		Translation: glm.Vec3f{10, 10, 0},
		Scale:       glm.Vec3f{5, 1},
	}

	// origin is exactly at the Translation value
	pos = tr.Affine2().Transform(glm.Vec3f{0, 0, 1})
	require.Equal(t, glm.Vec3f{10, 10, 1}, pos)

	pos = tr.Affine2().Transform(glm.Vec3f{0, 1, 1})
	require.Equal(t, glm.Vec3f{10, 11, 1}, pos)

	pos = tr.Affine2().Transform(glm.Vec3f{1, 0, 1})
	require.Equal(t, glm.Vec3f{15, 10, 1}, pos)

	// now rotate 90 deg
	tr.Rotation = glm.DegToRad(90)

	pos = tr.Affine2().Transform(glm.Vec3f{0, 1, 1})
	require.Equal(t, glm.Vec3f{9, 10, 1}, pos)

	pos = tr.Affine2().Transform(glm.Vec3f{1, 0, 1})
	require.Equal(t, glm.Vec3f{10, 15, 1}, pos)
}

func TestGlobalTransform_Mul(t *testing.T) {
	var pos glm.Vec3f

	tr := GlobalTransform{
		Translation: glm.Vec3f{10, 10, 0},
		Scale:       glm.Vec3f{5, 1},
	}

	// origin is exactly at the Translation value
	pos = tr.Mul(TransformFromXY(0, 0)).Affine2().Transform(glm.Vec3f{0, 0, 1})
	require.Equal(t, glm.Vec3f{10, 10, 1}, pos)

	pos = tr.Mul(TransformFromXY(0, 1)).Affine2().Transform(glm.Vec3f{0, 0, 1})
	require.Equal(t, glm.Vec3f{10, 11, 1}, pos)

	pos = tr.Mul(TransformFromXY(1, 0)).Affine2().Transform(glm.Vec3f{0, 0, 1})
	require.Equal(t, glm.Vec3f{15, 10, 1}, pos)

	// now rotate 90 deg
	tr.Rotation = glm.DegToRad(90)

	pos = tr.Mul(TransformFromXY(0, 1)).Affine2().Transform(glm.Vec3f{0, 0, 1})
	require.Equal(t, glm.Vec3f{9, 10, 1}, pos)

	pos = tr.Mul(TransformFromXY(1, 0)).Affine2().Transform(glm.Vec3f{0, 0, 1})
	require.Equal(t, glm.Vec3f{10, 15, 1}, pos)
}
