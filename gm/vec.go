package gm

import (
	"fmt"
	"math"
)

type ScalarTypes interface {
	float32 | float64 | int32
}

type Vec32 = vec[float32]
type Vec64 = vec[float64]

type Vec = Vec64

var VecOne = Vec{X: 1, Y: 1}

type IVec = vec[int32]

func VecOf[S int32 | float32 | float64](x, y S) vec[S] {
	return vec[S]{X: x, Y: y}
}

type vec[S int32 | float32 | float64] struct {
	X, Y S
}

func (v vec[S]) Add(other vec[S]) vec[S] {
	v.X += other.X
	v.Y += other.Y
	return v
}

func (v vec[S]) Sub(other vec[S]) vec[S] {
	v.X -= other.X
	v.Y -= other.Y
	return v
}

func (v vec[S]) Mul(scalar S) vec[S] {
	v.X *= scalar
	v.Y *= scalar
	return v
}

func (v vec[S]) MulEach(other vec[S]) vec[S] {
	v.X *= other.X
	v.Y *= other.Y
	return v
}

func (v vec[S]) DivEach(other vec[S]) vec[S] {
	v.X /= other.X
	v.Y /= other.Y
	return v
}

func (v vec[S]) String() string {
	return fmt.Sprintf("vec(x=%v, y=%v)", v.X, v.Y)
}

func (v vec[S]) Normalized() vec[S] {
	length := v.Length()
	v.X /= length
	v.Y /= length
	return v
}

func (v vec[S]) Length() S {
	return S(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}
