package byke

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalSystemParamStateValue(t *testing.T) {
	world := NewWorld()

	var run int
	system := func(l *Local[int]) {
		switch run {
		case 0:
			require.Equal(t, 0, l.Value)
			l.Value = 10

		case 1:
			require.Equal(t, 10, l.Value)
			l.Value = 20

		case 2:
			require.Equal(t, 20, l.Value)
		}
	}

	world.RunSystem(system)

	run = 1
	world.RunSystem(system)

	run = 2
	world.RunSystem(system)
}

func BenchmarkSystemParamState_Local(b *testing.B) {
	b.ReportAllocs()

	world := NewWorld()

	system := func(l *Local[int]) {}
	cachedSystem := AsCachedSystem(system)

	for b.Loop() {
		world.RunSystem(cachedSystem)
	}
}
