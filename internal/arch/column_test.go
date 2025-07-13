package arch

import (
	"math/rand/v2"
	"testing"
)

func BenchmarkTypedColumn_Get(b *testing.B) {
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

	var dummy Tick

	// get the row
	for b.Loop() {
		for row := range 1000 {
			componentValue := velocities.Get(Row(row))
			dummy += componentValue.Added
		}
	}
}
