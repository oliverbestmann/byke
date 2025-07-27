package gm

import (
	"fmt"
	"image"
)

type Rect struct {
	Min, Max Vec
}

func RectWithPoints(a, b Vec) Rect {
	return Rect{
		Min: Vec{
			X: min(a.X, b.X),
			Y: min(a.Y, b.Y),
		},
		Max: Vec{
			X: max(a.X, b.X),
			Y: max(a.Y, b.Y),
		},
	}
}

func RectWithSize(size Vec) Rect {
	return Rect{
		Min: VecZero,
		Max: size,
	}
}

func RectWithOriginAndSize(origin, size Vec) Rect {
	return Rect{
		Min: origin,
		Max: origin.Add(size),
	}
}

func RectWithCenterAndSize(center, size Vec) Rect {
	half := size.Mul(0.5)
	return Rect{
		Min: center.Sub(half),
		Max: center.Add(half),
	}
}

func (r Rect) Center() Vec {
	return r.Min.Add(r.Max).Mul(0.5)
}

func (r Rect) Size() Vec {
	return r.Max.Sub(r.Min)
}

func (r Rect) TopLeft() Vec {
	return r.Min
}

func (r Rect) TopRight() Vec {
	return Vec{
		X: r.Max.X,
		Y: r.Min.Y,
	}
}

func (r Rect) BottomLeft() Vec {
	return Vec{
		X: r.Min.X,
		Y: r.Max.Y,
	}
}

func (r Rect) BottomRight() Vec {
	return r.Max
}

func (r Rect) Translate(offset Vec) Rect {
	return Rect{
		Min: r.Min.Add(offset),
		Max: r.Max.Add(offset),
	}
}

func (r Rect) Contains(p Vec) bool {
	return r.Min.X <= p.X && p.X <= r.Max.X &&
		r.Min.Y <= p.Y && p.Y <= r.Max.Y
}

func (r Rect) ToImageRectangle() image.Rectangle {
	return image.Rectangle{
		Min: r.Min.ToImagePoint(),
		Max: r.Max.ToImagePoint(),
	}
}

func (r Rect) String() string {
	return fmt.Sprintf("Rect(min=%s, max=%s)", r.Min, r.Max)
}
