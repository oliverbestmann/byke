package arch

import "slices"

type ArchetypeGraph struct {
	transitions map[ArchetypeTransition]*Archetype
}

func (a *ArchetypeGraph) NextWith(current *Archetype, componentType *ComponentType) (*Archetype, bool) {
	tr := ArchetypeTransition{
		Archetype:     current,
		ComponentType: componentType,
		IsInsert:      true,
	}

	if next, ok := a.transitions[tr]; ok {
		return next, false
	}

	// get the target archetype by adding the componentType
	types := slices.Clone(current.Types)
	types = append(types, componentType)

	// build the new archetype if needed
	return a.insertTransition(tr, types), true
}

func (a *ArchetypeGraph) NextWithout(current *Archetype, componentType *ComponentType) (*Archetype, bool) {
	tr := ArchetypeTransition{
		Archetype:     current,
		ComponentType: componentType,
		IsInsert:      false,
	}

	if next, ok := a.transitions[tr]; ok {
		return next, false
	}

	// get the target archetype by removing the componentType
	var types []*ComponentType
	for _, ty := range current.Types {
		if ty != componentType {
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

	archetype := LookupArchetype(types)
	a.transitions[tr] = archetype

	return archetype
}

type ArchetypeTransition struct {
	Archetype     *Archetype
	ComponentType *ComponentType
	IsInsert      bool
}
