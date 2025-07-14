package byke

import (
	"errors"
	"github.com/oliverbestmann/byke/internal/set"
	"iter"
	"slices"
)

type Schedule struct {
	lookup     map[SystemId]*preparedSystem
	systems    []*preparedSystem
	systemSets []*SystemSet
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

func (s *Schedule) addSystemSet(systemSet *SystemSet) error {
	s.systemSets = append(s.systemSets, systemSet)
	return s.updateSystemOrdering()
}

func (s *Schedule) updateSystemOrdering() error {
	var configs []*systemConfig
	for _, system := range s.lookup {
		configs = append(configs, &system.systemConfig)
	}

	// calculate ordering
	ordering, err := topologicalSystemOrder(configs, s.systemSets)
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

func topologicalSystemOrder(systems []*systemConfig, knownSystemSets []*SystemSet) ([]SystemId, error) {
	// graph and in-degree count for topological sorting
	graph := map[SystemId][]SystemId{}
	inDegree := map[SystemId]int{}

	// make a lookup table so we can easily find all systems within a set
	reverseSystemSets := map[*SystemSet][]SystemId{}
	for _, system := range systems {
		for systemSet := range system.SystemSets.Values() {
			if !slices.Contains(knownSystemSets, systemSet) {
				continue
			}

			reverseSystemSets[systemSet] = append(reverseSystemSets[systemSet], system.Id)
		}
	}

	// build a set of reachable node ids
	var nodes set.Set[SystemId]
	for _, sys := range systems {
		nodes.Insert(sys.Id)
		for b := range sys.Before.Values() {
			nodes.Insert(b)
		}

		for a := range sys.After.Values() {
			nodes.Insert(a)
		}
	}

	// initialize graph and in-degree map
	for node := range nodes.Values() {
		graph[node] = []SystemId{}
		inDegree[node] = 0
	}

	// build graph
	for _, sys := range systems {
		for before := range sys.Before.Values() {
			graph[sys.Id] = append(graph[sys.Id], before)
			inDegree[before]++
		}

		for after := range sys.After.Values() {
			graph[after] = append(graph[after], sys.Id)
			inDegree[sys.Id]++
		}
	}

	// add extra edges for systems in sets
	for systemSet := range reverseSystemSets {
		// add one edge "systemSet -> beforeSet" for each system combination in both sets
		for _, beforeSet := range systemSet.before {
			for from, to := range cross(reverseSystemSets[systemSet], reverseSystemSets[beforeSet]) {
				if !slices.Contains(graph[from], to) {
					graph[from] = append(graph[from], to)
					inDegree[to] += 1
				}
			}
		}

		// add one edge "afterSet -> systemSet" for each system combination in both sets
		for _, afterSet := range systemSet.after {
			for from, to := range cross(reverseSystemSets[afterSet], reverseSystemSets[systemSet]) {
				if !slices.Contains(graph[from], to) {
					graph[from] = append(graph[from], to)
					inDegree[to] += 1
				}
			}
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

func cross(lhs, rhs []SystemId) iter.Seq2[SystemId, SystemId] {
	return func(yield func(l, r SystemId) bool) {
		for _, l := range lhs {
			for _, r := range rhs {
				if !yield(l, r) {
					return
				}
			}
		}
	}
}
