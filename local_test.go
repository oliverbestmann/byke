package byke

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocal(t *testing.T) {
	w := NewWorld()

	var countSeen int

	local := func(count *Local[int], other *Local[int]) {
		count.Value += 1
		countSeen = count.Value

		require.Equal(t, 0, other.Value)
	}

	w.AddSystems(Update, local)

	w.RunSchedule(Update)
	w.RunSchedule(Update)

	require.Equal(t, 2, countSeen)
}
