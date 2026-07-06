package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// KeyIsJustPressed returns a system predicate that runs a system only in the frame when
// the specified key is first pressed (edge detection).
func KeyIsJustPressed(key vyn.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsJustPressed(key)
	})
}

// KeyIsJustReleased returns a system predicate that runs a system only in the frame when
// the specified key is released (edge detection).
func KeyIsJustReleased(key vyn.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsJustReleased(key)
	})
}

// KeyIsPressed returns a system predicate that runs a system every frame the specified key
// is being held down.
func KeyIsPressed(key vyn.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsPressed(key)
	})
}

// MouseButtonIsJustPressed returns a system predicate that runs a system only in the frame when
// the specified mouse button is first pressed (edge detection).
func MouseButtonIsJustPressed(button vyn.MouseButton) byke.Systems {
	return byke.System(func(buttons MouseButtons) bool {
		return buttons.IsJustPressed(button)
	})
}

// MouseButtonIsJustReleased returns a system predicate that runs a system only in the frame when
// the specified mouse button is released (edge detection).
func MouseButtonIsJustReleased(button vyn.MouseButton) byke.Systems {
	return byke.System(func(buttons MouseButtons) bool {
		return buttons.IsJustReleased(button)
	})
}

// MouseButtonIsPressed returns a system predicate that runs a system every frame the specified
// mouse button is being held down.
func MouseButtonIsPressed(button vyn.MouseButton) byke.Systems {
	return byke.System(func(buttons MouseButtons) bool {
		return buttons.IsPressed(button)
	})
}
