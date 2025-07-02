package arch

import (
	"fmt"
	"iter"
	"sync"
)

type Storage struct {
	entityToArchetype map[EntityId]*Archetype
	archetypes        []*Archetype
	graph             ArchetypeGraph
}

func NewStorage() *Storage {
	return &Storage{
		entityToArchetype: map[EntityId]*Archetype{},
		archetypes:        nil,
	}
}

func (s *Storage) Spawn(entityId EntityId) {
	if _, exists := s.entityToArchetype[entityId]; exists {
		panic(fmt.Sprintf("entity %s already exists", entityId))
	}

	// put entity into empty archetype
	archetype := LookupArchetype(nil)

	if len(s.archetypes) == 0 {
		// store the empty archetype
		s.archetypes = append(s.archetypes, archetype)
	}

	// add entity to the archetype
	archetype.Insert(entityId, nil)

	// remember where we put the entity
	s.entityToArchetype[entityId] = archetype
}

func (s *Storage) Despawn(entityId EntityId) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return false
	}

	archetype.Remove(entityId)
	return true
}

func (s *Storage) InsertComponent(entityId EntityId, component ErasedComponent, tick uint64) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	componentType := component.ComponentType()
	if archetype.ContainsType(componentType) {
		archetype.ReplaceComponentValue(entityId, ComponentValue{
			Changed: tick,
			Hash:    componentType.MaybeHashOf(component),
			Value:   component,
		})

		return
	}

	// we need to move to a new archetype
	newArchetype, created := s.graph.NextWith(archetype, componentType)
	if created {
		s.archetypes = append(s.archetypes, newArchetype)
	}

	// transfer our entity
	archetype.TransferTo(newArchetype, entityId, ComponentValue{
		Added:   tick,
		Changed: tick,
		Hash:    componentType.MaybeHashOf(component),
		Value:   component,
	})

	// and update the entityToArchetype
	s.entityToArchetype[entityId] = newArchetype
}

func (s *Storage) RemoveComponent(entityId EntityId, componentType *ComponentType) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	if !archetype.ContainsType(componentType) {
		// entity does not have the component in question
		return false
	}

	// we need to move to a new archetype
	newArchetype, created := s.graph.NextWithout(archetype, componentType)
	if created {
		s.archetypes = append(s.archetypes, newArchetype)
	}

	// and transfer our entity
	archetype.TransferTo(newArchetype, entityId)

	return true
}

func (s *Storage) archetypeIterForQuery(q *Query) iter.Seq[*Archetype] {
	return func(yield func(*Archetype) bool) {
		for _, archetype := range s.archetypes {
			if !q.MatchesArchetype(archetype) {
				continue
			}

			if !yield(archetype) {
				return
			}
		}
	}
}

func (s *Storage) Get(entityId EntityId) (EntityRef, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return EntityRef{}, false
	}

	return archetype.Get(entityId)
}

func (s *Storage) GetWithQuery(q *Query, entityId EntityId) (EntityRef, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return EntityRef{}, false
	}

	if !q.MatchesArchetype(archetype) {
		return EntityRef{}, false
	}

	entity, ok := archetype.Get(entityId)
	if !ok {
		panic("archetype does not contain entity")
	}

	if !q.Matches(entity) {
		return EntityRef{}, false
	}

	return entity, true
}

var componentValueSlices = sync.Pool{
	New: func() any {
		// returning the slice directly would allocate when converting to an interface.
		// to not have this allocation, we need to return a pointer type here.
		var slice []ComponentValue
		return &slice
	},
}

// IterQuery returns an iterator over entity refs matching the given query.
// The EntityRef.Components field is only valid until the next EntityRef is produced.
func (s *Storage) IterQuery(q *Query) iter.Seq[EntityRef] {
	archetypesIter := s.archetypeIterForQuery(q)

	return func(yield func(EntityRef) bool) {
		// get a ComponentValue slice from the scratch pool
		// to minimize allocations
		scratch := componentValueSlices.Get().(*[]ComponentValue)
		defer func() { componentValueSlices.Put(scratch) }()

		for archetype := range archetypesIter {
			it := archetype.Iter(scratch)

			for it.More() {
				entity := it.Next()

				if !q.Matches(entity) {
					continue
				}

				if !yield(entity) {
					return
				}
			}
		}
	}
}
