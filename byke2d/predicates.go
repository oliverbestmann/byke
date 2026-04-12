package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/vyn"
)

func KeyIsJustPressed(key vyn.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsJustPressed(key)
	})
}

func KeyIsJustReleased(key vyn.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsJustReleased(key)
	})
}

func KeyIsPressed(key vyn.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsPressed(key)
	})
}

func MouseButtonIsJustPressed(button vyn.MouseButton) byke.Systems {
	return byke.System(func(buttons MouseButtons) bool {
		return buttons.IsJustPressed(button)
	})
}

func MouseButtonIsJustReleased(button vyn.MouseButton) byke.Systems {
	return byke.System(func(buttons MouseButtons) bool {
		return buttons.IsJustReleased(button)
	})
}

func MouseButtonIsPressed(button vyn.MouseButton) byke.Systems {
	return byke.System(func(buttons MouseButtons) bool {
		return buttons.IsPressed(button)
	})
}
