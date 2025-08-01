package spoke

import (
	"math/rand/v2"
	"testing"
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
		X, Y float32
		Z    float32

		// this forces the type to not be trivially hashable
		_ [0]string
	}

	velocities := ComponentTypeOf[Velocity]().MakeColumn()

	// append a random row
	for range 1000 {
		velocities.Append(1, &Velocity{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()})
	}

	b.ReportAllocs()
	b.ResetTimer()

	var n byte
	for b.Loop() {
		n += 1
		velocities.CheckChanged(Tick(2))
	}
}

func BenchmarkColumn_DirtyCheck(b *testing.B) {
	type Velocity struct {
		ComparableComponent[Velocity]
		X, Y float32
		Z    float32
	}

	velocities := ComponentTypeOf[Velocity]().MakeColumn()

	// append a random row
	for range 1000 {
		velocities.Append(1, &Velocity{X: rand.Float32(), Y: rand.Float32(), Z: rand.Float32()})
	}

	b.ReportAllocs()
	b.ResetTimer()

	var n byte
	for b.Loop() {
		n += 1
		velocities.CheckChanged(Tick(2))
	}
}
