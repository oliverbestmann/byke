package ecs

import (
	"github.com/stretchr/testify/require"
	"slices"
	"testing"
)

type Position struct {
	Component[Position]
	X, Y int
}

type Velocity struct {
	Component[Velocity]
	X, Y float64
}

type Player struct {
	Component[Player]
}

type Enemy struct {
	Component[Enemy]
}

var _ = ValidateComponent[Position]()
var _ = ValidateComponent[Velocity]()
var _ = ValidateComponent[Player]()
var _ = ValidateComponent[Enemy]()

func buildSimpleWorld() World {
	w := NewWorld()

	w.Spawn(w.ReserveEntityId(), []AnyComponent{
		Name("Player"),
		Player{},
		Position{},
		Velocity{},
	})

	w.Spawn(w.ReserveEntityId(), []AnyComponent{
		Name("Tree"),
		Position{},
	})

	w.Spawn(w.ReserveEntityId(), []AnyComponent{
		Name("Enemy"),
		Enemy{},
		Position{},
		Velocity{},
	})

	return w
}

func requireCallback(t *testing.T, fn func(allGood func())) {
	t.Helper()

	var called bool
	fn(func() { called = true })
	require.True(t, called)
}

func TestRunSystemWithQuery(t *testing.T) {
	w := buildSimpleWorld()

	t.Run("query with immutable component", func(t *testing.T) {
		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[Position]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with mutable component", func(t *testing.T) {
		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[*Position]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with optional immutable component", func(t *testing.T) {
		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[Option[Player]]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with optional mutable component", func(t *testing.T) {
		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[OptionMut[Player]]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with struct (immutable)", func(t *testing.T) {
		type MoveableItem struct {
			Position Position
			Velocity Velocity
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 2)
			})
		})
	})

	t.Run("query with struct (mutable)", func(t *testing.T) {
		type MoveableItem struct {
			Velocity Velocity
			Position *Position
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 2)
			})
		})
	})

	t.Run("query with struct (immutable, option)", func(t *testing.T) {
		type MoveableItem struct {
			Position Position
			Velocity Velocity
			Player   Option[Player]
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 2)
			})
		})
	})

	t.Run("query with struct (immutable, OptionMut)", func(t *testing.T) {
		type MoveableItem struct {
			Position Position
			Velocity OptionMut[Velocity]
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)

				for item := range q.Items() {
					value, ok := item.Velocity.Get()
					if ok {
						value.X = 1
					}
				}
			})
		})

		w.RunSystem(func(q Query[Velocity]) {
			for item := range q.Items() {
				require.Equal(t, 1.0, item.X, "velocity must have been updated")
			}
		})
	})

	t.Run("query with struct (immutable, has)", func(t *testing.T) {
		type MoveableItem struct {
			Position Position
			Velocity Velocity
			Player   Has[Player]
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 2)
			})
		})
	})
}

func BenchmarkWorld_RunSystem(b *testing.B) {
	type X struct {
		Component[X]
		Value int
	}

	type Y struct {
		Component[Y]
		Value int
	}

	w := NewWorld()

	w.RunSystem(func(c *Commands) {
		for range 2000 {
			c.Spawn(X{Value: 1}, Y{Value: 2})
		}
	})

	type Values struct {
		Name Option[Name]
		X    X
	}

	schedule := &Schedule{}
	w.AddSystems(schedule, func(q Query[Values]) {
		for range q.Items() {
			// do nothing
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		w.RunSchedule(schedule)
	}
}
