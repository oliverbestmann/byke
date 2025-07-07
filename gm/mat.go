package gm

import "math"

// Mat describes a 2d matrix of float64 values in row major order.
type Mat struct {
	XAxis, YAxis Vec
}

func IdentityMat() Mat {
	return Mat{
		XAxis: Vec{X: 1, Y: 0},
		YAxis: Vec{X: 0, Y: 1},
	}
}

// ScaleMat returns a matrix that scales a Vec.
func ScaleMat(scale Vec) Mat {
	return Mat{
		XAxis: Vec{scale.X, 0},
		YAxis: Vec{0, scale.Y},
	}
}

// RotationMat returns a rotation matrix that rotates
// a Vec clockwise by the given angle
func RotationMat(angle Rad) Mat {
	sin, cos := math.Sincos(float64(angle))

	return Mat{
		XAxis: Vec{cos, -sin},
		YAxis: Vec{sin, cos},
	}
}

func (m Mat) Transform(vec Vec) Vec {
	return Vec{
		X: m.XAxis.X*vec.X + m.XAxis.Y*vec.Y,
		Y: m.YAxis.X*vec.X + m.YAxis.Y*vec.Y,
	}
}

func (m Mat) Mul(n Mat) Mat {
	return Mat{
		XAxis: Vec{
			X: m.XAxis.X*n.XAxis.X + m.XAxis.Y*n.YAxis.X,
			Y: m.XAxis.X*n.XAxis.Y + m.XAxis.Y*n.YAxis.Y,
		},
		YAxis: Vec{
			X: m.YAxis.X*n.XAxis.X + m.YAxis.Y*n.YAxis.X,
			Y: m.YAxis.X*n.XAxis.Y + m.YAxis.Y*n.YAxis.Y,
		},
	}
}

func (m Mat) Inverse() Mat {
	f := 1 / (m.XAxis.X*m.YAxis.Y - m.XAxis.Y*m.YAxis.X)
	return Mat{
		XAxis: Vec{
			X: f * m.YAxis.Y,
			Y: f * -m.XAxis.Y,
		},
		YAxis: Vec{
			X: f * -m.YAxis.X,
			Y: f * m.XAxis.X,
		},
	}
}
