package byke

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
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
	t.Run("different systems", func(t *testing.T) {
		a := asSystemConfig(a).Id
		b := asSystemConfig(b).Id
		c := asSystemConfig(c).Id

		require.NotEqual(t, a, b)
		require.NotEqual(t, a, c)
		require.NotEqual(t, b, c)
	})

	t.Run("same system", func(t *testing.T) {
		a0 := asSystemConfig(a).Id
		a1 := asSystemConfig(a).Id
		a2 := asSystemConfig(a).Id

		require.Equal(t, a0, a1)
		require.Equal(t, a0, a2)
	})

}

func TestSystemIdWithClosure(t *testing.T) {
	a := asSystemConfig(makeSystem(1)).Id
	b := asSystemConfig(makeSystem(2)).Id
	require.NotEqual(t, a, b)
}

func TestSystemIdWithGeneric(t *testing.T) {
	a0 := asSystemConfig(gen[int]).Id
	a1 := asSystemConfig(gen[int]).Id
	require.Equal(t, a0, a1)

	b := asSystemConfig(gen[float32]).Id
	require.NotEqual(t, a0, b)
}

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

func gen[X any]() {
	ty := reflect.TypeFor[X]()
	fmt.Sprintln(ty)
}
