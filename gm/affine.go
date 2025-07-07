package gm

type Affine struct {
	Matrix      Mat
	Translation Vec
}

func IdentityAffine() Affine {
	return Affine{
		Matrix: IdentityMat(),
	}
}

func (a Affine) Transform(point Vec) Vec {
	return a.Matrix.Transform(point).Add(a.Translation)
}

func (a Affine) Mul(other Affine) Affine {
	return Affine{
		Matrix:      a.Matrix.Mul(other.Matrix),
		Translation: a.Matrix.Transform(other.Translation).Add(a.Translation),
	}
}

func (a Affine) Inverse() Affine {
	mat := a.Matrix.Inverse()
	translation := mat.Transform(a.Translation).Mul(-1)
	return Affine{
		Matrix:      mat,
		Translation: translation,
	}
}
