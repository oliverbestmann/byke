package byke

import (
	"errors"
	"github.com/oliverbestmann/byke/internal/set"
)

type Schedule struct {
	lookup  map[SystemId]*preparedSystem
	systems []*preparedSystem
}

func NewSchedule() *Schedule {
	return &Schedule{
		lookup: map[SystemId]*preparedSystem{},
	}
}

func (s *Schedule) addSystem(system *preparedSystem) error {
	if _, exists := s.lookup[system.Id]; exists {
		return errors.New("system already exists")
	}

	s.lookup[system.Id] = system
	return s.updateSystemOrdering()
}

func (s *Schedule) updateSystemOrdering() error {
	var configs []SystemConfig
	for _, system := range s.lookup {
		configs = append(configs, system.SystemConfig)
	}

	// calculate ordering
	ordering, err := topologicalSystemOrder(configs)
	if err != nil {
		return err
	}

	// recreate list of ordered systems
	s.systems = s.systems[:0]

	for _, id := range ordering {
		system, ok := s.lookup[id]
		if !ok {
			continue
		}

		s.systems = append(s.systems, system)
	}

	return nil
}

type preparedSystem struct {
	SystemConfig
	LastRun   Tick
	RawSystem func()
}

type systemOrdering struct {
	Before SystemId
	After  SystemId
}

type systemConstraint struct {
	// A predicate to check if the system should be run
	Predicate func() bool
}

func topologicalSystemOrder(systems []SystemConfig) ([]SystemId, error) {
	// graph and in-degree count for topological sorting
	graph := map[SystemId][]SystemId{}
	inDegree := map[SystemId]int{}

	// Ensure all nodes are in the graph
	var nodes set.Set[SystemId]
	for _, sys := range systems {
		nodes.Insert(sys.Id)
		for b := range sys.before.Values() {
			nodes.Insert(b)
		}

		for a := range sys.after.Values() {
			nodes.Insert(a)
		}
	}

	// Initialize graph and in-degree map
	for node := range nodes.Values() {
		graph[node] = []SystemId{}
		inDegree[node] = 0
	}

	// build graph
	for _, sys := range systems {
		for before := range sys.before.Values() {
			graph[sys.Id] = append(graph[sys.Id], before)
			inDegree[before]++
		}

		for after := range sys.after.Values() {
			graph[after] = append(graph[after], sys.Id)
			inDegree[sys.Id]++
		}
	}

	// topological sort using Kahn's algorithm
	var queue []SystemId
	for node, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, node)
		}
	}

	var result []SystemId
	for idx := 0; idx < len(queue); idx++ {
		curr := queue[idx]
		result = append(result, curr)

		for _, neighbor := range graph[curr] {
			inDegree[neighbor]--

			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// check for cycles
	if len(result) != nodes.Len() {
		return nil, errors.New("cycle detected or unresolved dependencies")
	}

	return result, nil
}
