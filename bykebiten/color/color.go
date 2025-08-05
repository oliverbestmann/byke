package color

var White = RGB(1, 1, 1)
var Black = RGB(0, 0, 0)
var Transparent = RGBA(0, 0, 0, 0)

// Color is a non alpha pre-multiplied color value in Color space.
// A value of 1 indicates full color
type Color struct {
	R, G, B, A float32
}

func RGBA(r, g, b, a float32) Color {
	return Color{R: r, G: g, B: b, A: a}
}

func RGB(r, g, b float32) Color {
	return RGBA(r, g, b, 1.0)
}

func Gray(g float32) Color {
	return RGB(g, g, g)
}

func (c Color) WithAlpha(a float32) Color {
	c.A = a
	return c
}

func (c Color) RGBA() (r, g, b, a uint32) {
	const MAX = 0xffff

	r = uint32(clamp(c.R*c.A*MAX, 0, MAX))
	g = uint32(clamp(c.G*c.A*MAX, 0, MAX))
	b = uint32(clamp(c.B*c.A*MAX, 0, MAX))
	a = uint32(clamp(c.A*MAX, 0, MAX))

	return
}

func (c Color) PremultipliedValues() (float32, float32, float32, float32) {
	r := c.R * c.A
	g := c.G * c.A
	b := c.B * c.A
	return r, g, b, c.A
}

func (c Color) IsIdentity() bool {
	return c.R == 1 && c.G == 1 && c.B == 1 && c.A == 1
}

func clamp[T float32 | float64](value, min, max T) T {
	if value < min {
		return min
	}

	if value > max {
		return max
	}

	return value
}
