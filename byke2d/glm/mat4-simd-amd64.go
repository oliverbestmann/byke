//go:build !nosimd && amd64 && goexperiment.simd

package glm

import "simd/archsimd"

// Multiplies m by o in place and saves the result back to o
func mat4fMulAssign(m, o *mat4f) {
	// load all columns of m into memory
	mc0 := archsimd.LoadFloat32x4(&m[0])
	mc1 := archsimd.LoadFloat32x4(&m[1])
	mc2 := archsimd.LoadFloat32x4(&m[2])
	mc3 := archsimd.LoadFloat32x4(&m[3])

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

		resC.Store(&m[0])
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

		resC.Store(&m[1])
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

		resC.Store(&m[2])
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

		resC.Store(&m[3])
	}
}

//goland:noinspection DuplicatedCode
func mat4fTranslate(m *mat4f, x, y, z float32) {
	mc0 := archsimd.LoadFloat32x4(&m[0])
	mc1 := archsimd.LoadFloat32x4(&m[1])
	mc2 := archsimd.LoadFloat32x4(&m[2])
	mc3 := archsimd.LoadFloat32x4(&m[3])

	oX := archsimd.BroadcastFloat32x4(x)
	oY := archsimd.BroadcastFloat32x4(y)
	oZ := archsimd.BroadcastFloat32x4(z)

	resC := mc0.MulAdd(oX, mc3)
	resC = mc1.MulAdd(oY, resC)
	resC = mc2.MulAdd(oZ, resC)

	resC.Store(&m[3])
}

//goland:noinspection DuplicatedCode
func mat4fScale(m *mat4f, x, y, z float32) {
	oX := archsimd.BroadcastFloat32x4(x)
	oY := archsimd.BroadcastFloat32x4(y)
	oZ := archsimd.BroadcastFloat32x4(z)

	mc0 := archsimd.LoadFloat32x4(&m[0]).Mul(oX)
	mc1 := archsimd.LoadFloat32x4(&m[1]).Mul(oY)
	mc2 := archsimd.LoadFloat32x4(&m[2]).Mul(oZ)

	mc0.Store(&m[0])
	mc1.Store(&m[1])
	mc2.Store(&m[2])
}
