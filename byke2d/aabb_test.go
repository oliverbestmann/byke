package byke2d

import (
	"testing"

	"github.com/oliverbestmann/byke/byke2d/glm"
)

func TestAABBOutsideClip(t *testing.T) {
	// camera at the origin, looking down -z
	worldToClip := DefaultPerspectiveProjection.ToMat4f(glm.Vec2f{16, 9})
	box := AABB{Min: glm.Vec3f{-1, -1, -1}, Max: glm.Vec3f{1, 1, 1}}

	cases := []struct {
		name    string
		at      glm.Vec3f
		scale   float32
		outside bool
	}{
		{"in front", glm.Vec3f{0, 0, -10}, 1, false},
		{"behind", glm.Vec3f{0, 0, 10}, 1, true},
		{"left", glm.Vec3f{-50, 0, -10}, 1, true},
		{"above", glm.Vec3f{0, 50, -10}, 1, true},
		{"crossing the edge", glm.Vec3f{-12, 0, -10}, 3, false},
		{"around the camera", glm.Vec3f{0, 0, 0}, 5, false},
	}

	for _, c := range cases {
		model := glm.TranslationMat4f(c.at[0], c.at[1], c.at[2]).Mul(glm.ScaleMat4f(c.scale, c.scale, c.scale))
		if got := box.OutsideClip(worldToClip.Mul(model)); got != c.outside {
			t.Errorf("%s: got outside=%v", c.name, got)
		}
	}
}
