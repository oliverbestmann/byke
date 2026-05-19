//go:build !nosimd && amd64 && goexperiment.simd

package glm

import "simd/archsimd"

func mat4fMulAssign(m, o, res *mat4f) {
	// load columns of m
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

		resC.Store(&res[0])
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

		resC.Store(&res[1])
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

		resC.Store(&res[2])
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

		resC.Store(&res[3])
	}
}
