package byke2d

import "github.com/oliverbestmann/byke/byke2d/glm"

type MouseCursor struct {
	glm.Vec2f
}

type MouseCursorDelta struct {
	glm.Vec2f
}

func updateMouseCursorSystem(cursor *MouseCursor, delta *MouseCursorDelta, inputState InputState) {
	x := inputState.state.Mouse.CursorX
	y := inputState.state.Mouse.CursorY
	cursor.Vec2f = glm.Vec2f{x, y}

	dx := inputState.state.Mouse.DeltaX
	dy := inputState.state.Mouse.DeltaY
	delta.Vec2f = glm.Vec2f{dx, dy}
}
