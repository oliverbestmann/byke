//go:build goexperiment.simd

package glm

import (
	"simd/archsimd"
	"unsafe"
)

type vec4f struct {
	X, Y, Z, W float32
}

func (a *vec4f) Dot(b *vec4f) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}

type vec4fs struct {
	// vec archsimd.Float32x4
}

type mat4f [4]vec4f

func (m mat4f) Column(idx int) vec4f {
	return m[idx]
}

//goland:noinspection DuplicatedCode
func (m mat4f) Mul(o mat4f) mat4f {
	var res mat4f

	res[0].X = m[0].X*o[0].X + m[1].X*o[0].Y + m[2].X*o[0].Z + m[3].X*o[0].W
	res[0].Y = m[0].Y*o[0].X + m[1].Y*o[0].Y + m[2].Y*o[0].Z + m[3].Y*o[0].W
	res[0].Z = m[0].Z*o[0].X + m[1].Z*o[0].Y + m[2].Z*o[0].Z + m[3].Z*o[0].W
	res[0].W = m[0].W*o[0].X + m[1].W*o[0].Y + m[2].W*o[0].Z + m[3].W*o[0].W

	res[1].X = m[0].X*o[1].X + m[1].X*o[1].Y + m[2].X*o[1].Z + m[3].X*o[1].W
	res[1].Y = m[0].Y*o[1].X + m[1].Y*o[1].Y + m[2].Y*o[1].Z + m[3].Y*o[1].W
	res[1].Z = m[0].Z*o[1].X + m[1].Z*o[1].Y + m[2].Z*o[1].Z + m[3].Z*o[1].W
	res[1].W = m[0].W*o[1].X + m[1].W*o[1].Y + m[2].W*o[1].Z + m[3].W*o[1].W

	res[2].X = m[0].X*o[2].X + m[1].X*o[2].Y + m[2].X*o[2].Z + m[3].X*o[2].W
	res[2].Y = m[0].Y*o[2].X + m[1].Y*o[2].Y + m[2].Y*o[2].Z + m[3].Y*o[2].W
	res[2].Z = m[0].Z*o[2].X + m[1].Z*o[2].Y + m[2].Z*o[2].Z + m[3].Z*o[2].W
	res[2].W = m[0].W*o[2].X + m[1].W*o[2].Y + m[2].W*o[2].Z + m[3].W*o[2].W

	res[3].X = m[0].X*o[3].X + m[1].X*o[3].Y + m[2].X*o[3].Z + m[3].X*o[3].W
	res[3].Y = m[0].Y*o[3].X + m[1].Y*o[3].Y + m[2].Y*o[3].Z + m[3].Y*o[3].W
	res[3].Z = m[0].Z*o[3].X + m[1].Z*o[3].Y + m[2].Z*o[3].Z + m[3].Z*o[3].W
	res[3].W = m[0].W*o[3].X + m[1].W*o[3].Y + m[2].W*o[3].Z + m[3].W*o[3].W

	return res
}

//goland:noinspection DuplicatedCode
func (m mat4f) MulSimd(o mat4f) mat4f {
	// load columns of m
	mc0 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[0])))
	mc1 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[1])))
	mc2 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[2])))
	mc3 := archsimd.LoadFloat32x4((*[4]float32)(unsafe.Pointer(&m[3])))

	var res mat4f

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

	return res
}

func mat4Scale(x, y, z float32) mat4f {
	var m mat4f
	m[0].X = x
	m[1].Y = y
	m[2].Z = z
	m[3].W = 1
	return m
}
