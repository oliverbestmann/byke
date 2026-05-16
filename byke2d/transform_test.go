package byke2d

import (
	"testing"

	"github.com/oliverbestmann/pulse/glm"
	"github.com/stretchr/testify/require"
)

func TestGlobalTransform_Affine2(t *testing.T) {
	var pos glm.Vec4f

	tr := Transform{
		Translation: glm.Vec3f{10, 10, 0},
		Scale:       glm.Vec3f{5, 1, 1},
	}

	// origin is exactly at the Translation value
	pos = tr.Affine3().Transform(glm.Vec4f{0, 0, 0, 1})
	require.Equal(t, glm.Vec4f{10, 10, 0, 1}, pos)

	pos = tr.Affine3().Transform(glm.Vec4f{0, 1, 0, 1})
	require.Equal(t, glm.Vec4f{10, 11, 0, 1}, pos)

	pos = tr.Affine3().Transform(glm.Vec4f{1, 0, 0, 1})
	require.Equal(t, glm.Vec4f{15, 10, 0, 1}, pos)

	// now rotate 90 deg
	tr.Rotation = glm.DegToRad(90)

	pos = tr.Affine3().Transform(glm.Vec4f{0, 1, 0, 1})
	require.Equal(t, glm.Vec4f{9, 10, 0, 1}, pos)

	pos = tr.Affine3().Transform(glm.Vec4f{1, 0, 0, 1})
	require.Equal(t, glm.Vec4f{10, 15, 0, 1}, pos)
}

func TestGlobalTransform_Mul(t *testing.T) {
	var pos glm.Vec4f

	base := Transform{
		Translation: glm.Vec3f{10, 10, 0},
		Scale:       glm.Vec3f{5, 1, 1},
	}

	tr := GlobalTransform{
		Affine: base.Affine3(),
	}

	// origin is exactly at the Translation value
	pos = tr.Mul(TransformFromXY(0, 0)).Affine.Transform(glm.Vec4f{0, 0, 0, 1})
	require.Equal(t, glm.Vec4f{10, 10, 0, 1}, pos)

	pos = tr.Mul(TransformFromXY(0, 1)).Affine.Transform(glm.Vec4f{0, 0, 0, 1})
	require.Equal(t, glm.Vec4f{10, 11, 0, 1}, pos)

	pos = tr.Mul(TransformFromXY(1, 0)).Affine.Transform(glm.Vec4f{0, 0, 0, 1})
	require.Equal(t, glm.Vec4f{15, 10, 0, 1}, pos)

	// now rotate 90 deg
	base.Rotation = glm.DegToRad(90)
	tr.Affine = base.Affine3()

	pos = tr.Mul(TransformFromXY(0, 1)).Affine.Transform(glm.Vec4f{0, 0, 0, 1})
	require.Equal(t, glm.Vec4f{9, 10, 0, 1}, pos)

	pos = tr.Mul(TransformFromXY(1, 0)).Affine.Transform(glm.Vec4f{0, 0, 0, 1})
	require.Equal(t, glm.Vec4f{10, 15, 0, 1}, pos)
}
