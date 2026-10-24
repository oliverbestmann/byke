package byke

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
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
		runTest(
			t,
			asSystemConfigs(
				System(a).Before(b),
				System(c).Before(a),
			),

			systemIdsOf(c, a, b),
		)
	})

	t.Run("a, b, c", func(t *testing.T) {
		runTest(
			t,
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
	for range 1024 {
		var SetA, SetB, SetC *SystemSet

		runTest := func(systems []*systemConfig, expected []SystemId) {
			order, err := topologicalSystemOrder(systems, []*SystemSet{})
			require.NoError(t, err)
			require.Equal(t, expected, order)
		}

		SetA = &SystemSet{Name: "SetA"}
		SetB = &SystemSet{Name: "SetB"}
		SetC = &SystemSet{Name: "SetC"}

		SetA.Before(SetB)
		SetB.Before(SetC)

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

		runTest(
			asSystemConfigs(
				System(c).InSet(SetC),
				System(a).InSet(SetA),
			),

			systemIdsOf(a, c),
		)
	}
}

// systems for the tests of one system function in two schedules
var scheduleLog []string

func logA() { scheduleLog = append(scheduleLog, "a") }
func logB() { scheduleLog = append(scheduleLog, "b") }

func never() bool { return false }

func logLocal(local *Local[int]) {
	local.Value++
	scheduleLog = append(scheduleLog, fmt.Sprint(local.Value))
}

func TestSystemInTwoSchedulesUsesSetsOfEach(t *testing.T) {
	// without an ordering, the order depends on map iteration. repeat to catch it.
	for range 64 {
		scheduleLog = nil

		scheduleX, scheduleY := MakeScheduleId("X"), MakeScheduleId("Y")
		setX := &SystemSet{Name: "X"}
		setY1, setY2 := &SystemSet{Name: "Y1"}, &SystemSet{Name: "Y2"}
		setY1.Before(setY2)

		world := NewWorld()
		world.AddSystems(scheduleX, System(logA).InSet(setX))
		world.AddSystems(scheduleY, System(logB).InSet(setY2))
		world.AddSystems(scheduleY, System(logA).InSet(setY1))

		world.RunSchedule(scheduleY)
		require.Equal(t, []string{"a", "b"}, scheduleLog)
	}
}

func TestSystemInTwoSchedulesUsesPredicatesOfEach(t *testing.T) {
	scheduleLog = nil

	scheduleX, scheduleY := MakeScheduleId("X"), MakeScheduleId("Y")

	world := NewWorld()
	world.AddSystems(scheduleX, logA)
	world.AddSystems(scheduleY, System(logA).RunIf(never))

	world.RunSchedule(scheduleY)
	require.Empty(t, scheduleLog, "the predicate of schedule Y must stop the system")

	world.RunSchedule(scheduleX)
	require.Equal(t, []string{"a"}, scheduleLog, "schedule X has no predicate")
}

func TestSystemInTwoSchedulesHasOwnLocal(t *testing.T) {
	scheduleLog = nil

	scheduleX, scheduleY := MakeScheduleId("X"), MakeScheduleId("Y")

	world := NewWorld()
	world.AddSystems(scheduleX, logLocal)
	world.AddSystems(scheduleY, logLocal)

	world.RunSchedule(scheduleX)
	world.RunSchedule(scheduleY)
	require.Equal(t, []string{"1", "1"}, scheduleLog)
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
	fmt.Println(ty)
}
