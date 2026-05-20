package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

var taaaOffsets = [4]glm.Vec2f{
	{0.5, 0.3333333333333333},
	{0.25, 0.6666666666666666},
	{0.75, 0.1111111111111111},
	{0.125, 0.4444444444444444},
}

type TAA struct {
	byke.Component[TAA]
}
