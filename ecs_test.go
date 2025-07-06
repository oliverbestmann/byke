package byke

import (
	"github.com/stretchr/testify/require"
	"slices"
	"testing"
)

type Position struct {
	ComparableComponent[Position]
	X, Y int
}

type Velocity struct {
	ComparableComponent[Velocity]
	X, Y int
}

type Player struct {
	ComparableComponent[Player]
	Value int
}

type Enemy struct {
	ComparableComponent[Enemy]
}

var _ = ValidateComponent[Position]()
var _ = ValidateComponent[Velocity]()
var _ = ValidateComponent[Player]()
var _ = ValidateComponent[Enemy]()

func buildSimpleWorld() *World {
	w := NewWorld()

	w.Spawn(w.ReserveEntityId(), []ErasedComponent{
		Named("Player"),
		Player{},
		Position{X: 1},
		Velocity{X: 10},
	})

	w.Spawn(w.ReserveEntityId(), []ErasedComponent{
		Named("Tree"),
		Position{X: 2},
	})

	w.Spawn(w.ReserveEntityId(), []ErasedComponent{
		Named("Enemy"),
		Enemy{},
		Position{X: 3},
		Velocity{Y: 20},
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
	t.Run("query with immutable component", func(t *testing.T) {
		w := buildSimpleWorld()

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[Position]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)

				// can collect again)
				require.Len(t, slices.Collect(q.Items()), 3)

				// count is also valid
				require.Equal(t, 3, q.Count())
			})
		})
	})

	t.Run("query with mutable component", func(t *testing.T) {
		w := buildSimpleWorld()

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[*Position]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with optional immutable component", func(t *testing.T) {
		w := buildSimpleWorld()

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[Option[Player]]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with optional mutable component", func(t *testing.T) {
		w := buildSimpleWorld()

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[OptionMut[Player]]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 3)
			})
		})
	})

	t.Run("query with struct (immutable)", func(t *testing.T) {
		w := buildSimpleWorld()

		type MoveableItem struct {
			Position Position
			Velocity Velocity
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()
				require.Len(t, slices.Collect(q.Items()), 2)

				for item := range q.Items() {
					require.NotZero(t, item.Position.X)
				}
			})
		})
	})

	t.Run("query with struct (mutable)", func(t *testing.T) {
		w := buildSimpleWorld()

		type MoveableItem struct {
			Velocity Velocity
			Position *Position
		}

		requireCallback(t, func(allGood func()) {
			w.RunSystem(func(q Query[MoveableItem]) {
				allGood()

				for item := range q.Items() {
					require.NotZero(t, item.Position.X)
					item.Position.X = item.Velocity.X * 2
				}
			})
		})

		w.RunSystem(func(q Query[MoveableItem]) {
			for item := range q.Items() {
				require.Equal(t, item.Velocity.X*2, item.Position.X)
			}
		})
	})

	t.Run("query with struct (immutable, option)", func(t *testing.T) {
		w := buildSimpleWorld()

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
		w := buildSimpleWorld()

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
				require.Equal(t, 1, item.X, "velocity must have been updated")
			}
		})
	})

	t.Run("query with struct (immutable, has)", func(t *testing.T) {
		w := buildSimpleWorld()

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

func TestChangeDetection(t *testing.T) {
	t.Run("query with component", func(t *testing.T) {
		w := buildSimpleWorld()

		var runCount int
		var expectedCount int

		w.AddSystems(Update, func(q Query[Changed[Position]]) {
			require.Equal(t, expectedCount, q.Count())
			runCount += 1
		})

		// should trigger for all newly added components
		expectedCount = 3
		w.RunSchedule(Update)
		require.Equal(t, 1, runCount)

		// should not trigger again if no selection was made
		expectedCount = 0
		w.RunSchedule(Update)
		require.Equal(t, 2, runCount)

		// update one of the positions
		w.RunSystem(func(q Query[*Position]) {
			q.MustGet().X += 1
		})

		// we should now see a change to exactly one of the fields
		expectedCount = 1
		w.RunSchedule(Update)
		require.Equal(t, 3, runCount)
	})
}

func TestRelationships(t *testing.T) {
	makeWorld := func() (w *World, parentId, childId EntityId) {
		w = NewWorld()

		w.RunSystem(func(commands *Commands) {
			parentId = commands.Spawn().Id()

			childId = commands.Spawn(ChildOf{
				Parent: parentId,
			}).Id()
		})

		return
	}

	t.Run("insert with ChildOf", func(t *testing.T) {
		w, parentId, childId := makeWorld()

		type ParentItems struct {
			EntityId EntityId
			Children Children
		}

		// check that we can select the Children component
		w.RunSystem(func(q Query[ParentItems]) {
			require.EqualValues(t, 1, q.Count(), "expect to select one time")
			item := q.MustGet()

			require.Len(t, item.Children.Children(), 1)
			require.Equal(t, item.Children.Children()[0], childId)

			require.Equal(t, parentId, item.EntityId)
		})

		type ChildItems struct {
			EntityId EntityId
			ChildOf  ChildOf
		}

		// check that we can select the parent component
		w.RunSystem(func(q Query[ChildItems]) {
			require.EqualValues(t, 1, q.Count())
			item := q.MustGet()

			require.Equal(t, childId, item.EntityId)
			require.Equal(t, parentId, item.ChildOf.Parent)
		})
	})

	t.Run("remove ChildOf", func(t *testing.T) {
		w, _, childId := makeWorld()

		w.RunSystem(func(commands *Commands) {
			commands.Entity(childId).
				Update(RemoveComponent[ChildOf]())
		})

		w.RunSystem(func(q Query[Children]) {
			require.Equal(t, 1, q.Count())
			require.Empty(t, q.MustGet().Children())
		})
	})

	t.Run("despawn child component", func(t *testing.T) {
		w, _, childId := makeWorld()

		w.RunSystem(func(commands *Commands) {
			commands.Entity(childId).Despawn()
		})

		w.RunSystem(func(q Query[Children]) {
			require.Equal(t, 1, q.Count())
			require.Empty(t, q.MustGet().Children())
		})
	})

	t.Run("despawn hierarchy", func(t *testing.T) {
		w, parentId, _ := makeWorld()

		w.RunSystem(func(commands *Commands) {
			commands.Entity(parentId).Despawn()
		})

		require.Zero(t, w.storage.EntityCount())
	})
}

func BenchmarkWorld_RunSystem(b *testing.B) {
	type X struct {
		ComparableComponent[X]
		Value int
	}

	type Y struct {
		ComparableComponent[Y]
		Value int
	}

	w := NewWorld()

	w.RunSystem(func(c *Commands) {
		for idx := range 2000 {
			ec := c.Spawn(X{Value: 1}, Y{Value: 2})
			if idx%2 == 0 {
				ec.Update(InsertComponent(Named("Component")))
			}
		}
	})

	type Values struct {
		Name Option[Name]
		X    X
	}

	var schedule ScheduleId = &scheduleId{}
	w.AddSystems(schedule, func(q Query[Values]) {
		for item := range q.Items() {
			// do nothing
			_ = item
		}
	})

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		w.RunSchedule(schedule)
	}
}
