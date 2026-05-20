package byke2d

import "github.com/oliverbestmann/byke/byke2d/vyn"

type Keys struct {
	state vyn.KeysState
}

func (k *Keys) IsJustPressed(key vyn.Key) bool {
	return k.state.JustPressed[key]
}

func (k *Keys) IsJustReleased(key vyn.Key) bool {
	return k.state.JustReleased[key]
}

func (k *Keys) IsPressed(key vyn.Key) bool {
	return k.state.Pressed[key]
}

func (k *Keys) IsReleased(key vyn.Key) bool {
	return !k.state.Pressed[key]
}

type MouseButtons struct {
	state vyn.MouseState
}

func (b *MouseButtons) IsJustPressed(button vyn.MouseButton) bool {
	return b.state.JustPressed[button]
}

func (b *MouseButtons) IsJustReleased(button vyn.MouseButton) bool {
	return b.state.JustReleased[button]
}

func (b *MouseButtons) IsPressed(button vyn.MouseButton) bool {
	return b.state.Pressed[button]
}

func (b *MouseButtons) IsReleased(button vyn.MouseButton) bool {
	return b.state.Pressed[button]
}
