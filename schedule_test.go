package byke

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"testing"
)

func TestTopologicalSystemOrder(t *testing.T) {
	runTest := func(systems []SystemConfig, expected []SystemId) {
		order, err := topologicalSystemOrder(systems)
		require.NoError(t, err)
		require.Equal(t, expected, order)
	}

	runTest(
		asSystemConfigs(
			System(a).Before(b),
			System(c).Before(a),
		),

		systemIdsOf(c, a, b),
	)

	runTest(
		asSystemConfigs(
			System(a).Before(c),
			System(b).After(a),
			System(b).Before(c),
		),
		systemIdsOf(a, b, c),
	)

	runTest(
		asSystemConfigs(SystemChain(a, b, c)),
		systemIdsOf(a, b, c))

	runTest(
		asSystemConfigs(SystemChain(a, b, c), System(x).Before(c).After(b).After(a)),
		systemIdsOf(a, b, x, c))
}

func systemIdsOf(systems ...AnySystem) []SystemId {
	var ids []SystemId

	for _, system := range asSystemConfigs(systems...) {
		ids = append(ids, system.Id)
	}

	return ids
}

func TestSystemId(t *testing.T) {
	a := asSystemConfig(a).Id
	b := asSystemConfig(b).Id
	c := asSystemConfig(c).Id

	require.NotEqual(t, a, b)
	require.NotEqual(t, a, c)
	require.NotEqual(t, b, c)

	fmt.Println(a, b, c)
}

func TestSystemIdWithClosure(t *testing.T) {
	t.Skip("Not working yet with our current SystemId implementation")
	a := asSystemConfig(makeSystem(rand.Int())).Id
	b := asSystemConfig(makeSystem(rand.Int())).Id
	require.NotEqual(t, a, b)
}

//go:noinline
func makeSystem(param int) func() int {
	return func() int {
		return param
	}
}

func a() int {
	return 1
}

func b() int {
	return 2
}

func c() int {
	return 3
}

func x() int {
	return 4
}
