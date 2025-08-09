package byke

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// an event that indicates an explosion
type Explode string

type Object struct {
	Component[Object]
}

func TestTriggers(t *testing.T) {
	w := NewWorld()

	var observedTrigger On[Explode]

	explodeSystem := func(trigger On[Explode], commands *Commands) {
		observedTrigger = trigger
		commands.Entity(trigger.Target).Despawn()
	}

	var id EntityId
	w.RunSystem(func(commands *Commands) {
		id = commands.Spawn(Object{}).Observe(explodeSystem).Id()
	})

	require.Zero(t, observedTrigger)

	w.RunSystem(func(commands *Commands) {
		commands.Entity(id).Trigger(Explode("Boom"))
	})

	require.NotZero(t, observedTrigger)
	require.Equal(t, id, observedTrigger.Target)
	require.Equal(t, Explode("Boom"), observedTrigger.Event)

	w.RunSystem(func(q Query[EntityId]) {
		_, exists := q.Get(id)
		require.False(t, exists)
	})
}
