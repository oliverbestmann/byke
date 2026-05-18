//go:build amd64 && goexperiment.simd

package glm

import "unsafe"
import "simd/archsimd"

func mat4fMulSimd(m, o, res *mat4f) {
	// load columns of m
	mc0 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[0])))
	mc1 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[1])))
	mc2 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[2])))
	mc3 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[3])))

	{
		// column 0
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[0].X)
		oY := archsimd.BroadcastFloat32x4(o[0].Y)
		oZ := archsimd.BroadcastFloat32x4(o[0].Z)
		oW := archsimd.BroadcastFloat32x4(o[0].W)

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.Store((*[4]float32)(unsafe.Pointer(&res[0])))
	}

	{
		// column 1
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[1].X)
		oY := archsimd.BroadcastFloat32x4(o[1].Y)
		oZ := archsimd.BroadcastFloat32x4(o[1].Z)
		oW := archsimd.BroadcastFloat32x4(o[1].W)

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.Store((*[4]float32)(unsafe.Pointer(&res[1])))
	}

	{
		// column 2
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[2].X)
		oY := archsimd.BroadcastFloat32x4(o[2].Y)
		oZ := archsimd.BroadcastFloat32x4(o[2].Z)
		oW := archsimd.BroadcastFloat32x4(o[2].W)

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.Store((*[4]float32)(unsafe.Pointer(&res[2])))
	}

	{
		// column 3
		// load x of current o column and broadcast
		oX := archsimd.BroadcastFloat32x4(o[3].X)
		oY := archsimd.BroadcastFloat32x4(o[3].Y)
		oZ := archsimd.BroadcastFloat32x4(o[3].Z)
		oW := archsimd.BroadcastFloat32x4(o[3].W)

		resC := mc0.Mul(oX)
		resC = mc1.MulAdd(oY, resC)
		resC = mc2.MulAdd(oZ, resC)
		resC = mc3.MulAdd(oW, resC)

		resC.Store((*[4]float32)(unsafe.Pointer(&res[3])))
	}
}
