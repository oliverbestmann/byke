package byke2d

import (
	"math/rand/v2"
	"testing"

	"github.com/oliverbestmann/byke/byke2d/glm"
)

func TestFrustumOutsideBox(t *testing.T) {
	// camera at the origin, looking down -z
	frustum := FrustumFromClip(DefaultPerspectiveProjection.ToMat4f(glm.Vec2f{16, 9}))
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
		if got := frustum.OutsideBox(box.Transformed(model)); got != c.outside {
			t.Errorf("%s: got outside=%v", c.name, got)
		}
	}
}

// The world box is larger than the rotated mesh box, so the frustum test may keep a mesh the
// exact corner test culls, but must never cull a mesh that has a corner inside the view.
func TestFrustumNeverCullsVisibleBoxes(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	worldToClip := DefaultPerspectiveProjection.ToMat4f(glm.Vec2f{16, 9})
	frustum := FrustumFromClip(worldToClip)
	box := AABB{Min: glm.Vec3f{-1, 0, -0.5}, Max: glm.Vec3f{1, 3, 0.5}}

	for range 10000 {
		at := glm.Vec3f{rng.Float32()*60 - 30, rng.Float32()*20 - 10, rng.Float32()*60 - 40}
		model := glm.TranslationMat4f(at[0], at[1], at[2]).Mul(glm.RotationYMat4f(glm.Rad(rng.Float32() * 6.3)))

		if frustum.OutsideBox(box.Transformed(model)) && anyCornerInside(box, worldToClip.Mul(model)) {
			t.Fatalf("box at %v culled, but a corner is visible", at)
		}
	}
}

func anyCornerInside(a AABB, localToClip glm.Mat4f) bool {
	for idx := range 8 {
		corner := glm.Vec4f{a.Min[0], a.Min[1], a.Min[2], 1}
		if idx&1 != 0 {
			corner[0] = a.Max[0]
		}
		if idx&2 != 0 {
			corner[1] = a.Max[1]
		}
		if idx&4 != 0 {
			corner[2] = a.Max[2]
		}

		c := localToClip.Transform(corner)
		if c[3] > 0 && abs32(c[0]) <= c[3] && abs32(c[1]) <= c[3] {
			return true
		}
	}

	return false
}

func BenchmarkFrustumOutsideBox(b *testing.B) {
	frustum := FrustumFromClip(DefaultPerspectiveProjection.ToMat4f(glm.Vec2f{16, 9}))
	box := AABB{Min: glm.Vec3f{-1, 0, -0.5}, Max: glm.Vec3f{1, 3, 0.5}}
	center, half := box.Transformed(glm.TranslationMat4f(20, 0, -10))

	for b.Loop() {
		frustum.OutsideBox(center, half)
	}
}
