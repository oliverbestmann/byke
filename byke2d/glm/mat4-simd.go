//go:build !nosimd && goexperiment.simd && (amd64 || arm64 || wasm)

package glm

import "simd/archsimd"

// mat4fMulAssignSimd multiplies m by o and stores the result in m.
func mat4fMulAssignSimd(m, o *mat4f) {
	// load all columns of m into memory
	mc0 := archsimd.LoadFloat32x4Array(&m[0])
	mc1 := archsimd.LoadFloat32x4Array(&m[1])
	mc2 := archsimd.LoadFloat32x4Array(&m[2])
	mc3 := archsimd.LoadFloat32x4Array(&m[3])

	{
		// column 0
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[0][0])
		oY := archsimd.BroadcastFloat32x4(o[0][1])
		oZ := archsimd.BroadcastFloat32x4(o[0][2])
		oW := archsimd.BroadcastFloat32x4(o[0][3])

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.StoreArray(&m[0])
	}

	{
		// column 1
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[1][0])
		oY := archsimd.BroadcastFloat32x4(o[1][1])
		oZ := archsimd.BroadcastFloat32x4(o[1][2])
		oW := archsimd.BroadcastFloat32x4(o[1][3])

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.StoreArray(&m[1])
	}

	{
		// column 2
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[2][0])
		oY := archsimd.BroadcastFloat32x4(o[2][1])
		oZ := archsimd.BroadcastFloat32x4(o[2][2])
		oW := archsimd.BroadcastFloat32x4(o[2][3])

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.StoreArray(&m[2])
	}

	{
		// column 3
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[3][0])
		oY := archsimd.BroadcastFloat32x4(o[3][1])
		oZ := archsimd.BroadcastFloat32x4(o[3][2])
		oW := archsimd.BroadcastFloat32x4(o[3][3])

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.StoreArray(&m[3])
	}
}

//goland:noinspection DuplicatedCode
func mat4fTranslateSimd(m *mat4f, x, y, z float32) {
	mc0 := archsimd.LoadFloat32x4Array(&m[0])
	mc1 := archsimd.LoadFloat32x4Array(&m[1])
	mc2 := archsimd.LoadFloat32x4Array(&m[2])
	mc3 := archsimd.LoadFloat32x4Array(&m[3])

	oX := archsimd.BroadcastFloat32x4(x)
	oY := archsimd.BroadcastFloat32x4(y)
	oZ := archsimd.BroadcastFloat32x4(z)

	resC := mc0.MulAdd(oX, mc3)
	resC = mc1.MulAdd(oY, resC)
	resC = mc2.MulAdd(oZ, resC)

	resC.StoreArray(&m[3])
}

//goland:noinspection DuplicatedCode
func mat4fScaleSimd(m *mat4f, x, y, z float32) {
	oX := archsimd.BroadcastFloat32x4(x)
	oY := archsimd.BroadcastFloat32x4(y)
	oZ := archsimd.BroadcastFloat32x4(z)

	mc0 := archsimd.LoadFloat32x4Array(&m[0]).Mul(oX)
	mc1 := archsimd.LoadFloat32x4Array(&m[1]).Mul(oY)
	mc2 := archsimd.LoadFloat32x4Array(&m[2]).Mul(oZ)

	mc0.StoreArray(&m[0])
	mc1.StoreArray(&m[1])
	mc2.StoreArray(&m[2])
}

func mat4fTransformBoxSimd(m *mat4f, c, h Vec3f) (Vec3f, Vec3f) {
	mc0 := archsimd.LoadFloat32x4Array(&m[0])
	mc1 := archsimd.LoadFloat32x4Array(&m[1])
	mc2 := archsimd.LoadFloat32x4Array(&m[2])
	mc3 := archsimd.LoadFloat32x4Array(&m[3])

	center := mc0.MulAdd(archsimd.BroadcastFloat32x4(c.X), mc3)
	center = mc1.MulAdd(archsimd.BroadcastFloat32x4(c.Y), center)
	center = mc2.MulAdd(archsimd.BroadcastFloat32x4(c.Z), center)

	half := mc0.Abs().Mul(archsimd.BroadcastFloat32x4(h.X))
	half = mc1.Abs().MulAdd(archsimd.BroadcastFloat32x4(h.Y), half)
	half = mc2.Abs().MulAdd(archsimd.BroadcastFloat32x4(h.Z), half)

	return Vec3f{center.GetElem(0), center.GetElem(1), center.GetElem(2)},
		Vec3f{half.GetElem(0), half.GetElem(1), half.GetElem(2)}
}
