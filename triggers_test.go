package byke

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// an event that indicates an explosion
type Explode struct {
	EventTarget
	Value string
}

type Object struct {
	Component[Object]
}

func TestTriggers(t *testing.T) {
	w := NewWorld()

	var observedTrigger On[Explode]

	explodeSystem := func(trigger On[Explode], commands *Commands) {
		observedTrigger = trigger
		commands.Entity(trigger.Event.TargetEntityId()).Despawn()
	}

	var id EntityId
	w.RunSystem(func(commands *Commands) {
		id = commands.Spawn(Object{}).Observe(explodeSystem).Id()
	})

	require.Zero(t, observedTrigger)

	ev := Explode{EventTarget: EventTarget(id), Value: "Boom"}
	w.RunSystem(func(commands *Commands) {
		commands.Trigger(ev)
	})

	require.NotZero(t, observedTrigger)
	require.Equal(t, id, observedTrigger.Event.TargetEntityId())
	require.Equal(t, ev, observedTrigger.Event)

	w.RunSystem(func(q Query[EntityId]) {
		_, exists := q.Get(id)
		require.False(t, exists)
	})
}

func TestGlobalObserverSeesEntityEvent(t *testing.T) {
	w := NewWorld()

	var global, scoped []EntityId

	w.AddObserver(NewObserver(func(trigger On[Explode]) {
		global = append(global, trigger.Event.TargetEntityId())
	}))

	var first, second EntityId
	w.RunSystem(func(commands *Commands) {
		first = commands.Spawn(Object{}).Observe(func(trigger On[Explode]) {
			scoped = append(scoped, trigger.Event.TargetEntityId())
		}).Id()

		second = commands.Spawn(Object{}).Id()
	})

	w.RunSystem(func(commands *Commands) {
		commands.Trigger(Explode{EventTarget: EventTarget(first)})
		commands.Trigger(Explode{EventTarget: EventTarget(second)})
	})

	require.Equal(t, []EntityId{first, second}, global)
	require.Equal(t, []EntityId{first}, scoped)
}
