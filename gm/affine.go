package gm

// Affine represents an affine transformation. It consists of a Matrix that describes
// rotation and scale, as well as a Translation vector.
//
// Use IdentityAffine to build a new identity transformation.
type Affine struct {
	Matrix      Mat
	Translation Vec
}

// IdentityAffine returns the identity transformation.
func IdentityAffine() Affine {
	return Affine{
		Matrix: IdentityMat(),
	}
}

func (a Affine) Rotate(angle Rad) Affine {
	rot := Affine{Matrix: RotationMat(angle)}
	return a.Mul(rot)
}

func (a Affine) Scale(scale Vec) Affine {
	rot := Affine{Matrix: ScaleMat(scale)}
	return a.Mul(rot)
}

func (a Affine) Translate(translate Vec) Affine {
	rot := Affine{Matrix: IdentityMat(), Translation: translate}
	return a.Mul(rot)
}

// Transform applies the affine transform to the given point and returns
// the transformed point.
func (a Affine) Transform(point Vec) Vec {
	return a.Matrix.Transform(point).Add(a.Translation)
}

// TransformVec applies the transform to a vector. This is different from transforming
// a point in that it will not apply the translation component of the Affine transform.
// The vector will only be rotated and scaled.
func (a Affine) TransformVec(vec Vec) Vec {
	return a.Matrix.Transform(vec)
}

// Mul multiplies the affine transformation with another transformation.
// The effect of the resulting transformation is the same as transforming a
// point first by a and then by other.
func (a Affine) Mul(other Affine) Affine {
	return Affine{
		Matrix:      a.Matrix.Mul(other.Matrix),
		Translation: a.Matrix.Transform(other.Translation).Add(a.Translation),
	}
}

// Inverse returns the inverse of the Affine transformation.
// This method will panic if an inverse can not be calculated.
func (a Affine) Inverse() Affine {
	mat := a.Matrix.Inverse()
	translation := mat.Transform(a.Translation).Mul(-1)
	return Affine{
		Matrix:      mat,
		Translation: translation,
	}
}

// TryInverse returns the inverse of the Affine transformation if possible.
func (a Affine) TryInverse() (inverse Affine, ok bool) {
	mat, ok := a.Matrix.TryInverse()
	if !ok {
		return Affine{}, false
	}

	translation := mat.Transform(a.Translation).Mul(-1)
	inverse = Affine{
		Matrix:      mat,
		Translation: translation,
	}

	return inverse, true
}
