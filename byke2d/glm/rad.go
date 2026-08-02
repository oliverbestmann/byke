package glm

import (
	"github.com/chewxy/math32"
)

type Rad float32

func (r Rad) Float32() float32 {
	return float32(r)
}

func (r Rad) Sin() float32 {
	return math32.Sin(float32(r))
}

func (r Rad) Cos() float32 {
	return math32.Cos(float32(r))
}

func (r Rad) SinCos() (sin float32, cos float32) {
	return math32.Sincos(float32(r))
}
