package byke2d

import "github.com/oliverbestmann/byke/byke2d/glm"

type MouseCursor struct {
	glm.Vec2f

	// Touch is true if the last pointer event came from a touch screen
	Touch bool

	// Grabbed is true while the cursor is locked to the window, see PrimaryWindow.SetCursorGrab
	Grabbed bool
}

type MouseCursorDelta struct {
	glm.Vec2f
}

func updateMouseCursorSystem(cursor *MouseCursor, delta *MouseCursorDelta, inputState InputState) {
	x := inputState.state.Mouse.CursorX
	y := inputState.state.Mouse.CursorY
	cursor.Vec2f = glm.Vec2f{x, y}
	cursor.Touch = inputState.state.Mouse.Touch
	cursor.Grabbed = inputState.state.Mouse.Grabbed

	dx := inputState.state.Mouse.DeltaX
	dy := inputState.state.Mouse.DeltaY
	delta.Vec2f = glm.Vec2f{dx, dy}
}
