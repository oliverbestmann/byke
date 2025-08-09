package spoke

import (
	"testing"

	"github.com/stretchr/testify/require"
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

func TestAddRemove(t *testing.T) {
	s := NewStorage()
	s.Spawn(Tick(0), EntityId(1), []ErasedComponent{
		&Velocity{},
		&Position{},
	})

	require.True(t, s.HasComponent(EntityId(1), ComponentTypeOf[Velocity]()))

	s.Despawn(EntityId(1))
	require.False(t, s.HasComponent(EntityId(1), ComponentTypeOf[Velocity]()))

	s.Spawn(Tick(0), EntityId(1), []ErasedComponent{
		&Velocity{},
		&Position{},
	})

	require.True(t, s.HasComponent(EntityId(1), ComponentTypeOf[Velocity]()))
}

func TestStorage_OptimizeQuery(t *testing.T) {
	s := NewStorage()

	q := s.OptimizeQuery(Query{
		Fetch: []FetchComponent{
			{
				ComponentType: ComponentTypeOf[Velocity](),
				Optional:      false,
			},
		},
	})

	s.Spawn(Tick(0), EntityId(1), []ErasedComponent{
		&Velocity{},
		&Position{},
	})

	iter := s.IterQuery(q, QueryContext{})
	_, ok := iter.Next()
	require.True(t, ok)

	_, ok = iter.Next()
	require.False(t, ok)

	// spawn a new entry with a new archetype
	s.Spawn(Tick(1), EntityId(2), []ErasedComponent{
		&Velocity{},
	})

	// run the query again
	iter = s.IterQuery(q, QueryContext{})

	_, ok = iter.Next()
	require.True(t, ok)

	_, ok = iter.Next()
	require.True(t, ok)

	_, ok = iter.Next()
	require.False(t, ok)

	// force a realloc of the second archetype by adding 1000 items. afterwards we should have
	// exactly 1000 + 2 entities
	for i := range 1000 {
		s.Spawn(Tick(2), EntityId(100+i), []ErasedComponent{
			&Velocity{},
		})
	}

	// run the query again
	iter = s.IterQuery(q, QueryContext{})

	for range 1002 {
		_, ok := iter.Next()
		require.True(t, ok)
	}

	_, ok = iter.Next()
	require.False(t, ok)

}
