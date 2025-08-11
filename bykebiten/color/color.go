package color

var White = RGB(1, 1, 1)
var Black = RGB(0, 0, 0)
var Transparent = Color{}

// Color is a a alpha pre-multiplied color value in Color space.
// A value of 1 indicates full color
type Color struct {
	R, G, B, A float32
}

// RGBA returns a Color value from non premultiplied color components.
func RGBA(r, g, b, a float32) Color {
	return Color{R: r * a, G: g * a, B: b * a, A: a}
}

// PreRGBA returns a Color value from premultiplied color components.
func PreRGBA(r, g, b, a float32) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// RGB returns a Color value with alpha set to 1
func RGB(r, g, b float32) Color {
	return PreRGBA(r, g, b, 1.0)
}

// Gray returns a Color value representing the given gray value.
func Gray(gr float32) Color {
	return RGB(gr, gr, gr)
}

// ScaleAlpha multiplies all components using the given alpha value.
func (c Color) ScaleAlpha(a float32) Color {
	c.R *= a
	c.G *= a
	c.B *= a
	c.A *= a
	return c
}

func (c Color) RGBA() (r, g, b, a uint32) {
	const MAX = 0xffff

	r = uint32(clamp(c.R*MAX, 0, MAX))
	g = uint32(clamp(c.G*MAX, 0, MAX))
	b = uint32(clamp(c.B*MAX, 0, MAX))
	a = uint32(clamp(c.A*MAX, 0, MAX))

	return
}

// Values returns the premultiplied components of this Color value
func (c Color) Values() (float32, float32, float32, float32) {
	return c.R, c.G, c.B, c.A
}

// IsWhite returns true, if all color values (and alpha value) are one.
func (c Color) IsWhite() bool {
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
