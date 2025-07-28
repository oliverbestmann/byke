package gm

import (
	"fmt"
	"image"
	"math"
)

type ScalarTypes interface {
	float32 | float64 | int32
}

type Vec32 = VecType[float32]
type Vec64 = VecType[float64]

// Vec is a 2d vector of float64 values.
type Vec = Vec64

var VecZero = Vec{}
var VecOne = Vec{X: 1, Y: 1}

type IVec32 = VecType[int32]

type Scalar interface {
	int32 | int64 | float32 | float64
}

// VecOf returns a new vector for the given values x and y.
func VecOf[S Scalar](x, y S) VecType[S] {
	return VecType[S]{X: x, Y: y}
}

// VecSplat returns a new Vec with both values set to value.
func VecSplat[S Scalar](value S) VecType[S] {
	return VecType[S]{X: value, Y: value}
}

type VecType[S Scalar] struct {
	X, Y S
}

// Add adds two vectors and returns the sum.
func (v VecType[S]) Add(other VecType[S]) VecType[S] {
	v.X += other.X
	v.Y += other.Y
	return v
}

// Sub subtracts the other vector from v and returns the difference.
func (v VecType[S]) Sub(other VecType[S]) VecType[S] {
	v.X -= other.X
	v.Y -= other.Y
	return v
}

// Mul multiplies each component of the Vec with a scalar value.
func (v VecType[S]) Mul(scalar S) VecType[S] {
	v.X *= scalar
	v.Y *= scalar
	return v
}

// MulEach multiplies each component with the respecting component
// of the second vector.
func (v VecType[S]) MulEach(other VecType[S]) VecType[S] {
	v.X *= other.X
	v.Y *= other.Y
	return v
}

// DivEach divides each component of v with the respecting component
// of the second vector.
func (v VecType[S]) DivEach(other VecType[S]) VecType[S] {
	v.X /= other.X
	v.Y /= other.Y
	return v
}

// Length returns the euclidean norm (the length) of this vector. If you just
// want to compare the length of two vectors, you can reduce the number of math.Sqrt
// calls by using LengthSqr.
func (v VecType[S]) Length() S {
	return S(math.Sqrt(float64(v.LengthSqr())))
}

// LengthSqr returns the square of the euclidean norm of the vector.
func (v VecType[S]) LengthSqr() S {
	return v.X*v.X + v.Y*v.Y
}

// Normalized returns the normalized vector of v.
// Will panic, if the length of the vector is zero.
func (v VecType[S]) Normalized() VecType[S] {
	length := v.Length()
	if length == 0 {
		panic("vector is zero, can not normalize")
	}

	v.X /= length
	v.Y /= length
	return v
}

// NormalizedOrZero returns the normalized vector of v.
// If the input vector is VecZero, the result will also be VecZero.
func (v VecType[S]) NormalizedOrZero() VecType[S] {
	length := v.Length()
	if length == 0 {
		return VecType[S]{}
	}

	v.X /= length
	v.Y /= length
	return v
}

// DistanceTo returns the distance between the vector v and other.
func (v VecType[S]) DistanceTo(other VecType[S]) S {
	return other.Sub(v).Length()
}

// DistanceToSqr returns the squared distance between the vector v and other.
func (v VecType[S]) DistanceToSqr(other VecType[S]) S {
	return other.Sub(v).LengthSqr()
}

// VecTo returns the vector that points from v to other.
func (v VecType[S]) VecTo(other VecType[S]) VecType[S] {
	return other.Sub(v)
}

// ToImagePoint converts the instance into a image.Point. The values are truncated.
func (v VecType[S]) ToImagePoint() image.Point {
	return image.Point{
		X: int(v.X),
		Y: int(v.Y),
	}
}

// Angle returns the angle the vector is pointing at
func (v VecType[S]) Angle() Rad {
	return Rad(math.Atan2(float64(v.Y), float64(v.X)))
}

// Rotate rotates the
func (v VecType[S]) Rotate(angle Rad) VecType[S] {
	res := RotationMat(angle).Transform(v.AsVec())
	return VecType[S]{X: S(res.X), Y: S(res.Y)}
}

// Cross product of two vectors.
func (v VecType[S]) Cross(other VecType[S]) S {
	return v.X*other.Y - v.Y*other.X
}

// AsVec returns a Vec for v, converting the components to float64.
func (v VecType[S]) AsVec() Vec {
	return Vec{X: float64(v.X), Y: float64(v.Y)}
}

func (v VecType[S]) XY() (S, S) {
	return v.X, v.Y
}

func (v VecType[S]) String() string {
	return fmt.Sprintf("[%v, %v]", v.X, v.Y)
}
