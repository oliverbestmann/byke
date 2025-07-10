package arch

import (
	"fmt"
	"iter"
	"sync"
)

type Storage struct {
	entityToArchetype map[EntityId]*Archetype
	archetypes        ArchetypeGraph
}

func NewStorage() *Storage {
	return &Storage{
		entityToArchetype: map[EntityId]*Archetype{},
	}
}

func (s *Storage) Spawn(tick Tick, entityId EntityId) {
	if _, exists := s.entityToArchetype[entityId]; exists {
		panic(fmt.Sprintf("entity %s already exists", entityId))
	}

	// put entity into empty archetype
	archetype := s.archetypes.Lookup(nil)

	// add entity to the archetype
	archetype.Insert(tick, entityId, nil)

	// remember where we put the entity
	s.entityToArchetype[entityId] = archetype
}

func (s *Storage) Despawn(entityId EntityId) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return false
	}

	archetype.Remove(entityId)

	delete(s.entityToArchetype, entityId)

	return true
}

func (s *Storage) InsertComponent(tick Tick, entityId EntityId, component ErasedComponent) ErasedComponent {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	componentType := component.ComponentType()
	if archetype.ContainsType(componentType) {
		return archetype.ReplaceComponentValue(tick, entityId, component)
	}

	// we need to move to a new archetype
	newArchetype, _ := s.archetypes.NextWith(archetype, componentType)

	// transfer our entity
	newArchetype.Import(tick, archetype, entityId, component)

	// remove from the previous archetype
	archetype.Remove(entityId)

	// and update the index
	s.entityToArchetype[entityId] = newArchetype

	componentValue, ok := newArchetype.GetComponent(entityId, componentType)
	if !ok {
		panic("component we've just inserted is gone")
	}

	return componentValue.Value
}

func (s *Storage) RemoveComponent(tick Tick, entityId EntityId, componentType *ComponentType) (ErasedComponent, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	if !archetype.ContainsType(componentType) {
		// entity does not have the component in question
		return nil, false
	}

	componentValue, ok := archetype.GetComponent(entityId, componentType)
	if !ok {
		panic("component does not exist in archetype")
	}

	copyOfComponent := componentType.New()
	componentType.SetValue(copyOfComponent, componentValue.Value)

	// we need to move to a new archetype
	newArchetype, _ := s.archetypes.NextWithout(archetype, componentType)

	// import the entity
	newArchetype.Import(tick, archetype, entityId)

	// remove it from the previous archetype
	archetype.Remove(entityId)

	// update index
	s.entityToArchetype[entityId] = newArchetype

	return copyOfComponent, true
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

func (s *Storage) CheckChanged(tick Tick, types []*ComponentType) {
	for _, ty := range types {
		if !ty.Comparable {
			continue
		}

		for _, archetype := range s.archetypes.All() {
			if !archetype.ContainsType(ty) {
				continue
			}

			archetype.CheckChanged(tick, ty)
		}
	}
}

func (s *Storage) archetypeIterForQuery(q *Query) iter.Seq[*Archetype] {
	return func(yield func(*Archetype) bool) {
		for _, archetype := range s.archetypes.All() {
			if !q.MatchesArchetype(archetype) {
				continue
			}

			if !yield(archetype) {
				return
			}
		}
	}
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
		for archetype := range archetypesIter {
			it := archetype.Iter()

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

func (s *Storage) HasComponent(entityId EntityId, componentType *ComponentType) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic("entity does not exist")
	}

	return archetype.ContainsType(componentType)
}

func (s *Storage) GetComponent(entityId EntityId, componentType *ComponentType) (ComponentValue, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic("entity does not exist")
	}

	return archetype.GetComponent(entityId, componentType)
}

func (s *Storage) EntityCount() int {
	return len(s.entityToArchetype)
}
