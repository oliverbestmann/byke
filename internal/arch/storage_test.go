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
	s := NewStorage()

	s.Spawn(1)
	s.InsertComponent(1, &Position{X: 10}, 5)
	s.InsertComponent(1, &Velocity{X: 0}, 5)

	s.Spawn(2)
	s.InsertComponent(2, &Velocity{X: 0}, 6)

	query := &Query{
		LastRun: 6,
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

	for entity := range s.IterQuery(query) {
		spew.Dump(entity)
	}
}

func BenchmarkStorageIterQuery(b *testing.B) {
	s := NewStorage()

	s.Spawn(1)
	s.InsertComponent(1, &Position{X: 10}, 5)
	s.InsertComponent(1, &Velocity{X: 0}, 5)

	s.Spawn(2)
	s.InsertComponent(2, &Velocity{X: 0}, 6)

	query := &Query{
		LastRun: 6,
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
