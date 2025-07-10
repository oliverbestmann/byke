package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Keys struct {
	// TODO maybe implement ourselves?
}

func (k Keys) IsJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

func (k Keys) IsJustReleased(key ebiten.Key) bool {
	return inpututil.IsKeyJustReleased(key)
}

func (k Keys) IsPressed(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key)
}

func (k Keys) IsReleased(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key)
}

type MouseButtons struct {
}

func (b MouseButtons) IsJustPressed(button ebiten.MouseButton) bool {
	return inpututil.IsMouseButtonJustPressed(button)
}

func (b MouseButtons) IsJustReleased(button ebiten.MouseButton) bool {
	return inpututil.IsMouseButtonJustReleased(button)
}

func (b MouseButtons) IsPressed(button ebiten.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(button)
}

func (b MouseButtons) IsReleased(button ebiten.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(button)
}
