package spoke

import (
	"math/rand/v2"
	"testing"
)

func BenchmarkQueryMovement(b *testing.B) {
	type Position struct {
		ComparableComponent[Position]
		X, Y float64
	}

	type Velocity struct {
		ComparableComponent[Velocity]
		X, Y float64
	}

	type Acceleration struct {
		ComparableComponent[Acceleration]
		X, Y float64
	}

	type Enemy struct {
		Component[Enemy]
	}

	storage := NewStorage()

	for idx := range EntityId(1000) {
		storage.Spawn(0, idx)
		storage.InsertComponent(0, idx, &Position{})
		storage.InsertComponent(0, idx, &Velocity{})
		storage.InsertComponent(0, idx, &Acceleration{X: rand.Float64(), Y: rand.Float64()})

		if idx%2 == 0 {
			storage.InsertComponent(0, idx, &Enemy{})
		}
	}

	var qb QueryBuilder
	qb.FetchComponent(ComponentTypeOf[Position](), false)
	qb.FetchComponent(ComponentTypeOf[Velocity](), false)
	qb.FetchComponent(ComponentTypeOf[Acceleration](), false)

	query := storage.OptimizeQuery(qb.Build())

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		iter := storage.IterQuery(query, QueryContext{})
		for {
			entity, ok := iter.Next()
			if !ok {
				break
			}

			position := (*Position)(entity.GetAt(0))
			velocity := (*Velocity)(entity.GetAt(1))
			acceleration := (*Acceleration)(entity.GetAt(2))

			velocity.X += acceleration.X
			velocity.Y += acceleration.Y

			position.X += velocity.X
			position.Y += velocity.Y
		}
	}
}
