package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/byke"
)

type Shader struct {
	byke.Component[Shader]
	Shader *ebiten.Shader
}
