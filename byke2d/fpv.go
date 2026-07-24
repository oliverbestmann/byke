package byke2d

import (
	"cmp"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

var _ = byke.ValidateComponent[FirstPersonViewController]()

// FirstPersonViewController is a component that provides first-person camera controls
// with pitch and yaw rotation, and WASD movement with Q/E for vertical movement.
type FirstPersonViewController struct {
	byke.Component[FirstPersonViewController]

	// Pitch is the vertical rotation angle in radians.
	Pitch glm.Rad
	// Yaw is the horizontal rotation angle in radians.
	Yaw glm.Rad

	// Velocity is the movement speed in units per second. Defaults to 4m/s if not set.
	Velocity float32
}

func pluginFPV(app *byke.App) {
	app.AddSystems(byke.Update, fpvMoveSystem)
}

func fpvMoveSystem(
	vt byke.VirtualTime,
	keys Keys,

	items byke.Query[struct {
		FPV       *FirstPersonViewController
		Transform *Transform
	}],
) {
	for item := range items.Items() {
		if keys.IsPressed(vyn.KeyArrowLeft) {
			item.FPV.Yaw += glm.Rad(2 * vt.DeltaSecs)
		}

		if keys.IsPressed(vyn.KeyArrowRight) {
			item.FPV.Yaw -= glm.Rad(2 * vt.DeltaSecs)
		}

		if keys.IsPressed(vyn.KeyArrowUp) {
			item.FPV.Pitch += glm.Rad(2 * vt.DeltaSecs)
		}

		if keys.IsPressed(vyn.KeyArrowDown) {
			item.FPV.Pitch -= glm.Rad(2 * vt.DeltaSecs)
		}

		var move glm.Vec3f
		var moveAbsY float32

		velocity := cmp.Or(item.FPV.Velocity, 4.0)

		if keys.IsPressed(vyn.KeyA) {
			move[0] -= velocity * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyD) {
			move[0] += velocity * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyW) {
			move[2] -= velocity * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyS) {
			move[2] += velocity * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyQ) {
			// TODO this should be flipped
			moveAbsY -= velocity * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyE) {
			moveAbsY += velocity * vt.DeltaSecs
		}

		yaw := item.FPV.Yaw
		pitch := item.FPV.Pitch

		// apply new rotation
		item.Transform.Rotation = glm.RotationXQuat(pitch).Mul(glm.RotationYQuat(yaw))

		// transform the movement offset
		moveTransformed := item.Transform.
			Affine3().
			Transform(move.Extend(0))

		moveTransformed[1] += moveAbsY

		item.Transform.Translation = item.Transform.Translation.Add(moveTransformed.Truncate())
	}
}
