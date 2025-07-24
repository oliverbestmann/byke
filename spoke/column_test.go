package spoke

import (
	"math/rand/v2"
	"testing"
	"unsafe"
)

func BenchmarkColumn_Get1k(b *testing.B) {
	type Velocity struct {
		ComparableComponent[Velocity]
		X, Y float64
	}

	velocities := ComponentTypeOf[Velocity]().MakeColumn()

	// append a random row
	for range 1000 {
		velocities.Append(1, &Velocity{X: rand.Float64(), Y: rand.Float64()})
	}

	b.ReportAllocs()
	b.ResetTimer()

	var dummy bool

	// get the row
	for b.Loop() {
		for row := range 1000 {
			componentValue := velocities.Get(Row(row))
			dummy = dummy || componentValue != nil
		}
	}
}

func BenchmarkColumn_CheckChanges(b *testing.B) {
	type Velocity struct {
		ComparableComponent[Velocity]
		X, Y float64
	}

	velocities := ComponentTypeOf[Velocity]().MakeColumn()

	// append a random row
	for range 1000 {
		velocities.Append(1, &Velocity{X: rand.Float64(), Y: rand.Float64()})
	}

	b.ReportAllocs()
	b.ResetTimer()

	ComponentTypeOf[Velocity]().memcmp = false

	var n byte
	for b.Loop() {
		for idx := range 200 {
			*(*byte)(unsafe.Add(velocities.memory, (300+idx*2)*int(velocities.itemSize))) = n
		}

		n += 1

		velocities.CheckChanged(Tick(2))
	}
}

func BenchmarkColumn_DirtyCheck(b *testing.B) {
	type Velocity struct {
		ComparableComponent[Velocity]
		X, Y float64
	}

	velocities := ComponentTypeOf[Velocity]().MakeColumn()

	// append a random row
	for range 1000 {
		velocities.Append(1, &Velocity{X: rand.Float64(), Y: rand.Float64()})
	}

	b.ReportAllocs()
	b.ResetTimer()

	var n byte
	for b.Loop() {
		for idx := range 200 {
			*(*byte)(unsafe.Add(velocities.memory, (300+idx*2)*int(velocities.itemSize))) = n
		}

		n += 1

		velocities.memcheck(Tick(2))
	}
}
