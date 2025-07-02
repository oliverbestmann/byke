package arch

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

type Position struct {
	ComparableComponent[Position]
	X int
}

type Velocity struct {
	ComparableComponent[Velocity]
	X int
}

func TestStorage_All(t *testing.T) {
	var tick Tick = 1

	s := NewStorage()

	s.Spawn(tick, 1)
	s.InsertComponent(tick, 1, &Position{X: 10})
	s.InsertComponent(tick, 1, &Velocity{X: 0})

	tick += 1
	s.Spawn(tick, 2)
	s.InsertComponent(tick, 2, &Velocity{X: 0})

	tick += 1

	query := &Query{
		LastRun: tick,
		Fetch: []*ComponentType{
			ComponentTypeOf[Velocity](),
		},
		With: []*ComponentType{
			// ComponentTypeOf[Position](),
		},
		WithChanged: []*ComponentType{
			// ComponentTypeOf[Velocity](),
		},
		Without: []*ComponentType{
			ComponentTypeOf[Position](),
		},
	}

	for entity := range s.IterQuery(query) {
		spew.Dump(entity)
		entity.Components[0].Value.(*Velocity).X = 2
	}

	s.CheckChanged(7, []*ComponentType{ComponentTypeOf[Velocity]()})

	spew.Dump(s.Get(1))
}

func BenchmarkStorageIterQuery(b *testing.B) {
	var tick Tick = 5

	s := NewStorage()

	s.Spawn(tick, 1)
	s.InsertComponent(tick, 1, &Position{X: 10})
	s.InsertComponent(tick, 1, &Velocity{X: 0})

	tick += 1

	s.Spawn(tick, 2)
	s.InsertComponent(tick, 2, &Velocity{X: 0})

	tick += 1

	query := &Query{
		LastRun: tick,
		Fetch: []*ComponentType{
			ComponentTypeOf[Velocity](),
		},
		With: []*ComponentType{
			// ComponentTypeOf[Position](),
		},
		WithChanged: []*ComponentType{
			ComponentTypeOf[Velocity](),
		},
	}

	iter := s.IterQuery(query)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for entity := range iter {
			_ = entity
		}
	}
}
