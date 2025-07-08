package gm

import "math"

// Mat describes a 2d matrix of float64 values in row major order.
// The zero value is NOT the identity matrix. Use one of the initializer
// functions like IdentityMat, RotationMat or ScaleMat to get a new matrix.
type Mat struct {
	XAxis, YAxis Vec
}

// IdentityMat returns an identity matrix
func IdentityMat() Mat {
	return Mat{
		XAxis: Vec{X: 1, Y: 0},
		YAxis: Vec{X: 0, Y: 1},
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

// ScaleMat returns a matrix that scales a Vec.
func ScaleMat(scale Vec) Mat {
	return Mat{
		XAxis: Vec{scale.X, 0},
		YAxis: Vec{0, scale.Y},
	}
}

// Transform multiplies the matrix with the given vector.
// It returns the resulting vector.
func (m Mat) Transform(vec Vec) Vec {
	return Vec{
		X: m.XAxis.X*vec.X + m.XAxis.Y*vec.Y,
		Y: m.YAxis.X*vec.X + m.YAxis.Y*vec.Y,
	}
}

// Mul multiplies the matrix m with the matrix n and returns the
// result of the multiplication.
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

// Inverse calculates the inverse of the matrix. If the Determinant of the
// matrix is zero, the matrix is not invertible. In this case, this method will
// panic.
func (m Mat) Inverse() Mat {
	det := m.Determinant()
	if det == 0 {
		panic("matrix is not invertible")
	}

	f := 1 / det
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

// Determinant returns the determinant of the matrix
func (m Mat) Determinant() float64 {
	return m.XAxis.X*m.YAxis.Y - m.XAxis.Y*m.YAxis.X
}
