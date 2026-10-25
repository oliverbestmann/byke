package byke2d

import (
	"cmp"
	"math"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

var _ = byke.ValidateComponent[FirstPersonViewController]()

// FirstPersonViewController is a component that provides first-person camera controls
// with pitch and yaw rotation, and WASD movement with Q/E for vertical movement.
//
// It looks around with the arrow keys and with the mouse, after a click grabbed the cursor.
// On touch screens, dragging looks around and keeps turning for a bit after release.
type FirstPersonViewController struct {
	byke.Component[FirstPersonViewController]

	// Pitch is the vertical rotation angle in radians.
	Pitch glm.Rad
	// Yaw is the horizontal rotation angle in radians.
	Yaw glm.Rad

	// Velocity is the movement speed in units per second. Defaults to 4m/s if not set.
	Velocity float32

	lastCursor *glm.Vec2f

	// fling velocity in rad/s as (yaw, pitch), only used on touch screens
	fling glm.Vec2f
}

const (
	// mouse look speed in rad per pixel
	fpvMouseSensitivity = 0.003

	// how fast the fling velocity follows the drag speed, per second
	fpvFlingTracking = 20

	// how fast a fling slows down, per second
	fpvFlingFriction = 4
)

func pluginFPV(app *byke.App) {
	app.AddSystems(byke.Update, fpvMoveSystem)
}

// flingTurn tracks the drag speed and keeps turning with it after a touch is released.
// It returns the turn for this frame.
func (c *FirstPersonViewController) flingTurn(turn glm.Vec2f, dragging bool, dt float32) glm.Vec2f {
	if dragging {
		if dt > 0 {
			// follow the drag speed smoothly, so the fling uses the speed of the last few frames
			alpha := 1 - float32(math.Exp(float64(-fpvFlingTracking*dt)))
			c.fling = c.fling.Add(turn.Scale(1 / dt).Sub(c.fling).Scale(alpha))
		}

		return turn
	}

	// released: keep turning and slow down
	turn = c.fling.Scale(dt)
	c.fling = c.fling.Scale(float32(math.Exp(float64(-fpvFlingFriction * dt))))
	return turn
}

func fpvMoveSystem(
	vt byke.VirtualTime,
	keys Keys,
	buttons MouseButtons,
	cursor MouseCursor,
	screen ScreenSize,
	win *PrimaryWindow,

	items byke.Query[struct {
		FPV        *FirstPersonViewController
		Transform  *Transform
		Projection byke.Option[PerspectiveProjection]
	}],
) {
	for item := range items.Items() {
		// touch screens look by dragging, a mouse looks after a click grabbed the cursor
		if !cursor.Touch && !cursor.Grabbed && buttons.IsJustPressed(vyn.MouseButtonLeft) {
			win.SetCursorGrab(true)
		}

		// diff the cursor ourselves: MouseCursorDelta keeps its last value when the mouse stops.
		// a new touch starts somewhere else, so don't diff against the end of the last one.
		var delta glm.Vec2f
		if last := item.FPV.lastCursor; last != nil && !buttons.IsJustPressed(vyn.MouseButtonLeft) {
			delta = cursor.Vec2f.Sub(*last)
		}

		item.FPV.lastCursor = new(cursor.Vec2f)

		var turn glm.Vec2f

		switch {
		case cursor.Touch && screen.Vec2f[1] > 0:
			// mirrored, so the view follows the finger, and scaled so a drag over the full screen
			// height turns by the vertical fov. phones have many device pixels, a fixed rad/px turns way too fast there.
			fov := item.Projection.Or(DefaultPerspectiveProjection).Fov
			turn = delta.Scale(float32(fov) / screen.Vec2f[1])

			// the release frame may still carry the last drag movement
			dragging := buttons.IsPressed(vyn.MouseButtonLeft) || buttons.IsJustReleased(vyn.MouseButtonLeft)
			turn = item.FPV.flingTurn(turn, dragging, vt.DeltaSecs)

		case cursor.Grabbed:
			turn = delta.Scale(-fpvMouseSensitivity)
		}

		item.FPV.Yaw += glm.Rad(turn[0])
		item.FPV.Pitch += glm.Rad(turn[1])

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

		// do not look over the top, the view would flip
		item.FPV.Pitch = max(min(item.FPV.Pitch, glm.DegToRad(89)), glm.DegToRad(-89))

		yaw := item.FPV.Yaw
		pitch := item.FPV.Pitch

		// apply new rotation
		item.Transform.Rotation = glm.RotationYQuat(yaw).Mul(glm.RotationXQuat(pitch))

		// transform the movement offset
		moveTransformed := item.Transform.
			Affine3().
			Transform(move.Extend(0))

		moveTransformed[1] += moveAbsY

		item.Transform.Translation = item.Transform.Translation.Add(moveTransformed.Truncate())
	}
}
