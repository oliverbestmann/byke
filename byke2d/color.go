package byke2d

import (
	"math"

	"github.com/oliverbestmann/pulse/glm"
)

var ColorWhite = ColorLinearRGBA(1, 1, 1, 1)
var ColorBlack = ColorLinearRGBA(0, 0, 0, 1)
var ColorTransparent = ColorLinearRGBA(0, 0, 0, 0)

// Color is an a straight rgba color value with alpha in linear rgb color space.
// The default value of a Color value is fully opaque white.
type Color struct {
	r1, g1, b1, a1 float32
}

// ColorOf converts the linear rgb values from the given vector to a Color instance.
func ColorOf(color glm.Vec4f) Color {
	return ColorLinearRGBA(color[0], color[1], color[2], color[3])
}

// ColorLinearRGBA creates a new Color value from the given color values.
func ColorLinearRGBA(r, g, b, a float32) Color {
	return Color{
		r1: r - 1,
		g1: g - 1,
		b1: b - 1,
		a1: a - 1,
	}
}

// ColorSRGBA creates a Color value from non linear srgb encoded values. The color values
// will be transferred into linear rgb space.
// This is the usual color format on most devices.
// Use this if you picked a color from a jpeg image.
func ColorSRGBA(r, g, b, a float32) Color {
	r = degamma(r)
	g = degamma(g)
	b = degamma(b)

	return ColorLinearRGBA(r, g, b, a)
}

// ToVec returns a glm.Vec4f containing the components of this Color instance in
// linear rgb space.
func (c Color) ToVec() glm.Vec4f {
	return glm.Vec4f{
		c.r1 + 1,
		c.g1 + 1,
		c.b1 + 1,
		c.a1 + 1,
	}
}

func (c Color) ToWGPU() [4]float32 {
	return [4]float32{
		c.r1 + 1,
		c.g1 + 1,
		c.b1 + 1,
		c.a1 + 1,
	}
}

// Components returns the color components.
func (c Color) Components() (r, g, b, a float32) {
	return c.ToVec().XYZW()
}

// Scaled returns a new color with each component scaled by the corresponding
// component in the vector.
func (c Color) Scaled(vec glm.Vec4f) Color {
	return ColorOf(c.ToVec().Mul(vec))
}

// Alpha returns the alpha value of the color.
func (c Color) Alpha() float32 {
	return c.a1 + 1
}

func (c Color) Red() float32 {
	return c.r1 + 1
}

func (c Color) Green() float32 {
	return c.g1 + 1
}

func (c Color) Blue() float32 {
	return c.b1 + 1
}

// WithAlpha returns a new color with the alpha component set to the given value.
func (c Color) WithAlpha(alpha float32) Color {
	c.a1 = alpha - 1
	return c
}

func degamma(value float32) float32 {
	x := float64(value)

	// https://www.w3.org/TR/css-color-4/#color-conversion-code
	sign := math.Copysign(1, x)
	abs := math.Abs(x)
	if abs <= 0.04045 {
		return float32(x / 12.92)
	}

	return float32(sign * math.Pow((abs+0.055)/1.055, 2.4))
}
