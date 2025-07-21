package arch

import "slices"

type ArchetypeGraph struct {
	Archetypes
	transitions map[ArchetypeTransition]*Archetype
}

func (a *ArchetypeGraph) NextWith(current *Archetype, componentType *ComponentType) (*Archetype, bool) {
	tr := ArchetypeTransition{
		Archetype:      current,
		ComponentTypes: [8]*ComponentType{componentType},
		IsInsert:       true,
	}

	if next, ok := a.transitions[tr]; ok {
		return next, false
	}

	typeCount := 1

	// get the target archetype by adding the componentType
	types := slices.Clone(current.Types)
	types = append(types, tr.ComponentTypes[:typeCount]...)

	// build the new archetype if needed
	return a.insertTransition(tr, types), true
}

func (a *ArchetypeGraph) NextWithout(current *Archetype, componentType *ComponentType) (*Archetype, bool) {
	tr := ArchetypeTransition{
		Archetype:      current,
		ComponentTypes: [8]*ComponentType{componentType},
		IsInsert:       false,
	}

	if next, ok := a.transitions[tr]; ok {
		return next, false
	}

	typeCount := 1

	// get the target archetype by removing the componentTypes
	var types []*ComponentType
	for _, ty := range current.Types {
		if !slices.Contains(tr.ComponentTypes[:typeCount], ty) {
			types = append(types, ty)
		}
	}

	// build the new archetype if needed
	return a.insertTransition(tr, types), true
}

func (a *ArchetypeGraph) insertTransition(tr ArchetypeTransition, types []*ComponentType) *Archetype {
	if _, exists := a.transitions[tr]; exists {
		panic("archetype transition already exists")
	}

	if a.transitions == nil {
		a.transitions = map[ArchetypeTransition]*Archetype{}
	}

	archetype := a.Archetypes.Lookup(types)
	a.transitions[tr] = archetype

	return archetype
}

type ArchetypeTransition struct {
	Archetype      *Archetype
	ComponentTypes [8]*ComponentType
	IsInsert       bool
}
