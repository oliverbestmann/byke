package arch

import (
	"fmt"
	"iter"
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

	componentValue := newArchetype.GetComponent(entityId, componentType)
	if componentValue == nil {
		panic("component we've just inserted is gone")
	}

	return componentValue
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

	componentValue := archetype.GetComponent(entityId, componentType)
	if componentValue == nil {
		panic("component does not exist in archetype")
	}

	copyOfComponent := componentType.CopyOf(componentValue)

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

	_, fetches := archetype.IterForQuery(q, nil)
	entity.fetch = asFastSlice(fetches)

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

func (s *Storage) archetypeIterForQuery(q *Query) ArchetypeIter {
	return ArchetypeIter{
		archetypes: s.archetypes.All(),
		query:      q,
	}
}

type ArchetypeIter struct {
	archetypes []*Archetype
	query      *Query
}

func (it *ArchetypeIter) Next() *Archetype {
	for len(it.archetypes) > 0 {
		archetype := it.archetypes[0]
		it.archetypes = it.archetypes[1:]

		if len(archetype.entities) == 0 {
			continue
		}

		if it.query.MatchesArchetype(archetype) {
			return archetype
		}
	}

	return nil
}

// IterQuery returns an iterator over entity refs matching the given query.
// The EntityRef.Components field is only valid until the next EntityRef is produced.
func (s *Storage) IterQuery(q *Query, scratch []ColumnAccess) QueryIter {
	return QueryIter{
		Scratch:    scratch,
		archetypes: s.archetypeIterForQuery(q),
		query:      q,
	}
}

func (s *Storage) HasComponent(entityId EntityId, componentType *ComponentType) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic("entity does not exist")
	}

	return archetype.ContainsType(componentType)
}

func (s *Storage) EntityCount() int {
	return len(s.entityToArchetype)
}

type QueryIter struct {
	Scratch []ColumnAccess

	query      *Query
	archetypes ArchetypeIter
	entities   EntityIter
}

func (it *QueryIter) Next() (EntityRef, bool) {
	for {
		for it.entities.More() {
			entity := it.entities.Current()
			if it.query.Matches(entity) {
				return entity, true
			}
		}

		// no more entities in current entity iterator, move to the next one
		archetype := it.archetypes.Next()
		if archetype == nil {
			return EntityRef{}, false
		}

		it.entities, it.Scratch = archetype.IterForQuery(it.query, it.Scratch)
	}
}

func (it *QueryIter) AsSeq() iter.Seq[EntityRef] {
	return func(yield func(EntityRef) bool) {
		for {
			ref, ok := it.Next()
			if !ok {
				return
			}

			if !yield(ref) {
				return
			}
		}
	}
}
