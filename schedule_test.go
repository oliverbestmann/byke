package byke

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func systemIdsOf(systems ...AnySystem) []SystemId {
	var ids []SystemId

	for _, system := range systems {
		ids = append(ids, asSystemConfig(system).Id)
	}

	return ids
}

func TestSystemOrder(t *testing.T) {
	runTest := func(t *testing.T, systems []*systemConfig, expected []SystemId) {
		for _, system := range systems {
			fmt.Println(system.Id, system.Before, system.After)
		}

		order, err := topologicalSystemOrder(systems, nil)
		require.NoError(t, err)
		require.Equal(t, expected, order)
	}

	t.Run("c, a, b", func(t *testing.T) {
		runTest(t,
			asSystemConfigs(
				System(a).Before(b),
				System(c).Before(a),
			),

			systemIdsOf(c, a, b),
		)
	})

	t.Run("a, b, c", func(t *testing.T) {
		runTest(t,
			asSystemConfigs(
				System(a).Before(c),
				System(b).After(a),
				System(b).Before(c),
			),
			systemIdsOf(a, b, c),
		)
	})

	t.Run("a, b, c", func(t *testing.T) {
		runTest(t,
			asSystemConfigs(System(a, b, c).Chain()),
			systemIdsOf(a, b, c))
	})

	t.Run("a, b, x, c", func(t *testing.T) {
		runTest(t,
			asSystemConfigs(System(a, b, c).Chain(), System(x).Before(c).After(b).After(a)),
			systemIdsOf(a, b, x, c))
	})
}

func TestSystemOrderWithSets(t *testing.T) {
	var SetA, SetB *SystemSet

	runTest := func(systems []*systemConfig, expected []SystemId) {
		order, err := topologicalSystemOrder(systems, []*SystemSet{SetA, SetB})
		require.NoError(t, err)
		require.Equal(t, expected, order)
	}

	SetA = &SystemSet{}
	SetB = &SystemSet{}

	SetA.Before(SetB)

	runTest(
		asSystemConfigs(
			System(a, b).Chain().InSet(SetB),
			System(c).InSet(SetA),
		),

		systemIdsOf(c, a, b),
	)

	runTest(
		asSystemConfigs(
			System(b).InSet(SetB),
			System(a).InSet(SetA),
			System(c).After(b),
		),

		systemIdsOf(a, b, c),
	)
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
