package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

var _ = byke.ValidateComponent[FirstPersonViewController]()

type FirstPersonViewController struct {
	byke.Component[FirstPersonViewController]

	Pitch glm.Rad
	Yaw   glm.Rad
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
			item.FPV.Yaw -= glm.Rad(2 * vt.DeltaSecs)
		}

		if keys.IsPressed(vyn.KeyArrowRight) {
			item.FPV.Yaw += glm.Rad(2 * vt.DeltaSecs)
		}

		if keys.IsPressed(vyn.KeyArrowUp) {
			item.FPV.Pitch -= glm.Rad(2 * vt.DeltaSecs)
		}

		if keys.IsPressed(vyn.KeyArrowDown) {
			item.FPV.Pitch += glm.Rad(2 * vt.DeltaSecs)
		}

		var move glm.Vec3f

		if keys.IsPressed(vyn.KeyA) {
			move[0] -= 2 * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyD) {
			move[0] += 2 * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyW) {
			move[2] += 2 * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyS) {
			move[2] -= 2 * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyQ) {
			move[1] -= 2 * vt.DeltaSecs
		}

		if keys.IsPressed(vyn.KeyE) {
			move[1] += 2 * vt.DeltaSecs
		}

		yaw := item.FPV.Yaw
		pitch := item.FPV.Pitch

		// apply new rotation
		item.Transform.Rotation = glm.RotationXQuat(pitch).Mul(glm.RotationYQuat(yaw))

		// transform the movement offset
		moveTransformed := item.Transform.
			Affine3().
			Transform(move.Extend(0))

		item.Transform.Translation = item.Transform.Translation.Add(moveTransformed.Truncate())
	}
}
