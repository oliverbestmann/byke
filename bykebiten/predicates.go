package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
)

func KeyJustPressed(key ebiten.Key) byke.Systems {
	return byke.System(func(keys Keys) bool {
		return keys.IsJustPressed(key)
	})
}
