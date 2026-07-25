package byke2d

// Adapted from https://github.com/alltom/oklab

import (
	"github.com/chewxy/math32"
)

type Oklab struct {
	// Perceived lightness
	L float32
	// How green/red the color is
	A float32
	// How blue/yellow the color is
	B float32
}

type Oklch struct {
	// Perceived lightness
	L float32
	// Chroma
	C float32
	// Hue (in radians)
	H float32
}

// ToColor converts to linear sRGB.
// See https://bottosson.github.io/posts/oklab/
func (c Oklab) ToColor() Color {
	l_ := c.L + 0.3963377774*c.A + 0.2158037573*c.B
	m_ := c.L - 0.1055613458*c.A - 0.0638541728*c.B
	s_ := c.L - 0.0894841775*c.A - 1.2914855480*c.B

	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_

	r := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s

	return ColorLinearRGB(r, g, b)
}

// ToOklch converts to LCh, which is Oklab in polar.
func (c Oklab) ToOklch() Oklch {
	return Oklch{
		L: c.L,
		C: math32.Sqrt(c.A*c.A + c.B*c.B),
		H: math32.Atan2(c.B, c.A),
	}
}

// ToOklab convert to Oklab.
func (c Oklch) ToOklab() Oklab {
	return Oklab{
		L: c.L,
		A: c.C * math32.Cos(c.H),
		B: c.C * math32.Sin(c.H),
	}
}

// ToColor converts to linear sRGB.
func (c Oklch) ToColor() Color {
	return c.ToOklab().ToColor()
}
