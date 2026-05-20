package vyn

import "log/slog"

type UpdateInputState func() InputState

type KeysState struct {
	// the keys that are currently marked as "pressed"
	Pressed map[Key]bool

	// keys that where just pressed after the last call to nextTick()
	JustPressed map[Key]bool

	// keys that were just released after the last call to nextTick()
	JustReleased map[Key]bool
}

func (k *KeysState) press(key Key) {
	if !k.Pressed[key] {
		slog.Info("Key just pressed", slog.String("key", key.String()))

		setTrue(&k.Pressed, key)
		setTrue(&k.JustPressed, key)
	}
}

func (k *KeysState) release(key Key) {
	if k.Pressed[key] {
		setFalse(&k.Pressed, key)
		setTrue(&k.JustReleased, key)
	}
}

func (k *KeysState) nextTick() {
	clear(k.JustPressed)
	clear(k.JustReleased)
}

type MouseButton uint32

const MouseButtonLeft MouseButton = 0
const MouseButtonRight MouseButton = 1
const MouseButtonMiddle MouseButton = 2

type MouseState struct {
	CursorX, CursorY float32

	// recorded position since last tick
	DeltaX, DeltaY float32

	Pressed map[MouseButton]bool

	// mouse buttons that were just clicked after the last call to nextTick()
	JustPressed map[MouseButton]bool

	// mouse buttons that were just released after the last call to nextTick()
	JustReleased map[MouseButton]bool

	// we keep this around to handle delta calculation
	prevX, prevY float32
	tick         int
}

func (m *MouseState) press(button MouseButton) {
	setTrue(&m.Pressed, button)
	setTrue(&m.JustPressed, button)
}

func (m *MouseState) release(button MouseButton) {
	setFalse(&m.Pressed, button)
	setTrue(&m.JustReleased, button)
}

func (m *MouseState) position(x, y float32) {
	m.CursorX = x
	m.CursorY = y

	if m.tick > 1 {
		// only update delta starting after the first tick.
		m.DeltaX = x - m.prevX
		m.DeltaY = y - m.prevY
	}
}

func (m *MouseState) nextTick() {
	m.prevX = m.CursorX
	m.prevY = m.CursorY
	m.tick += 1

	clear(m.JustPressed)
	clear(m.JustReleased)
}

type InputState struct {
	Keys  KeysState
	Mouse MouseState
}

func (s *InputState) nextTick() {
	s.Keys.nextTick()
	s.Mouse.nextTick()
}

func setTrue[K comparable](m *map[K]bool, key K) {
	if *m == nil {
		*m = map[K]bool{}
	}

	(*m)[key] = true
}

func setFalse[K comparable](m *map[K]bool, key K) {
	if *m == nil {
		*m = map[K]bool{}
	}

	(*m)[key] = false
}
