package spoke

import (
	"github.com/stretchr/testify/require"
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

	s.Spawn(tick, 1, []ErasedComponent{
		&Position{X: 10}, &Velocity{X: 0},
	})

	tick += 1
	s.Spawn(tick, 2, []ErasedComponent{
		&Velocity{X: 1},
	})

	tick += 1

	query := Query{
		Fetch: []FetchComponent{
			{
				ComponentType: ComponentTypeOf[Velocity](),
			},
		},
		Filters: []Filter{
			{
				Without: ComponentTypeOf[Position](),
			},
		},
	}

	q := s.OptimizeQuery(query)
	iter := s.IterQuery(q, QueryContext{LastRun: tick})
	for entity := range iter.AsSeq() {
		value := entity.Get(ComponentTypeOf[Velocity]())
		value.(*Velocity).X = 2
	}

	s.CheckChanged(7, q, []*ComponentType{ComponentTypeOf[Velocity]()})

	{
		entity, _ := s.Get(1)
		tick := entity.Changed(ComponentTypeOf[Velocity]())
		require.Equal(t, Tick(1), tick)
	}

	{
		entity, _ := s.Get(2)
		tick := entity.Changed(ComponentTypeOf[Velocity]())
		require.Equal(t, Tick(7), tick)
	}
}

func BenchmarkStorageIterQuery(b *testing.B) {
	var tick Tick = 5

	s := NewStorage()

	s.Spawn(tick, 1, nil)
	s.InsertComponent(tick, 1, &Position{X: 10})
	s.InsertComponent(tick, 1, &Velocity{X: 0})

	tick += 1

	s.Spawn(tick, 2, nil)
	s.InsertComponent(tick, 2, &Velocity{X: 0})

	tick += 1

	query := Query{
		Fetch: []FetchComponent{
			{
				ComponentType: ComponentTypeOf[Velocity](),
			},
		},
		Filters: []Filter{
			{
				Without: ComponentTypeOf[Position](),
			},
		},
	}

	iter := s.IterQuery(s.OptimizeQuery(query), QueryContext{LastRun: tick})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for {
			entity, ok := iter.Next()
			if !ok {
				break
			}

			_ = entity
		}
	}
}
