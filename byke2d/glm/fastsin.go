package glm

import (
	"golang.org/x/mobile/exp/f32"
)

func fastSincos(r Rad) (float32, float32) {
	return fastSin(r), fastCos(r)
}

func fastSin(r Rad) float32 {
	return f32.Sin(float32(r))
}

func fastCos(r Rad) float32 {
	return f32.Cos(float32(r))
}
