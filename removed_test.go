package byke

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemovedComponents(t *testing.T) {
	w := NewWorld()

	enemyId := w.Spawn([]ErasedComponent{&Enemy{}})

	var expectedId EntityId
	w.AddSystems(Update, func(c RemovedComponents[Enemy]) {
		ids := slices.Collect(c.Read())
		if expectedId == 0 {
			require.Len(t, ids, 0)
		} else {
			require.Len(t, ids, 1)
			require.Equal(t, expectedId, ids[0])
		}
	})

	w.RunSchedule(Update)

	w.RunSystem(func(commands *Commands) {
		commands.Entity(enemyId).Despawn()
	})

	// detect the removal for the enemy component
	expectedId = enemyId
	w.RunSchedule(Update)

	// now zero
	expectedId = 0
	w.RunSchedule(Update)

}
