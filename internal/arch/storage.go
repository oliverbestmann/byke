package arch

import (
	"fmt"
	"iter"
)

type Storage struct {
	entityToArchetype map[EntityId]*Archetype
	archetypes        ArchetypeGraph
	queryCache        queryCache
}

func NewStorage() *Storage {
	storage := &Storage{
		entityToArchetype: map[EntityId]*Archetype{},
	}

	storage.queryCache.archetypes = &storage.archetypes

	return storage
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

func (s *Storage) InsertComponents(tick Tick, entityId EntityId, components []ErasedComponent) {
	prevArchetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	newArchetype := prevArchetype

	var created, anyCreated bool
	for _, component := range components {
		componentType := component.ComponentType()
		if newArchetype.ContainsType(componentType) {
			continue
		}

		// we need to move to a new archetype
		newArchetype, created = s.archetypes.NextWith(newArchetype, componentType)
		anyCreated = anyCreated || created
	}

	if anyCreated {
		s.handleNewArchetype(newArchetype)
	}

	if newArchetype == prevArchetype {
		// no change in archetypes, update in existing archetype
		for idx, component := range components {
			components[idx] = prevArchetype.ReplaceComponentValue(tick, entityId, component)
		}
	}

	// transfer our entity
	newArchetype.Import(tick, prevArchetype, entityId, components...)

	// remove from the previous archetype
	prevArchetype.Remove(entityId)

	// and update the index
	s.entityToArchetype[entityId] = newArchetype

	for idx, component := range components {
		componentValue := newArchetype.GetComponent(entityId, component.ComponentType())
		if componentValue == nil {
			panic("component we've just inserted is gone")
		}

		components[idx] = componentValue
	}
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
	newArchetype, created := s.archetypes.NextWith(archetype, componentType)
	if created {
		s.handleNewArchetype(newArchetype)
	}

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
	newArchetype, created := s.archetypes.NextWithout(archetype, componentType)
	if created {
		s.handleNewArchetype(newArchetype)
	}

	// import the entity
	newArchetype.Import(tick, archetype, entityId)

	// remove it from the previous archetype
	archetype.Remove(entityId)

	// update index
	s.entityToArchetype[entityId] = newArchetype

	return copyOfComponent, true
}

func (s *Storage) handleNewArchetype(newArchetype *Archetype) {
	doOptimize := func() { s.queryCache.Optimize(newArchetype) }

	// a new archetype was created,
	// we might need to re-optimize some queries
	doOptimize()

	// we register a callback to re-optimize all queries that are looking at data
	// of one of the columns to update any changed pointers
	for _, column := range newArchetype.columns {
		column.OnGrow = doOptimize
	}
}

func (s *Storage) Get(entityId EntityId) (EntityRef, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return EntityRef{}, false
	}

	return archetype.Get(entityId)
}

func (s *Storage) GetWithQuery(q *Query, ctx QueryContext, entityId EntityId) (EntityRef, bool) {
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

	if !q.IsArchetypeOnly {
		if !q.Matches(ctx, entity) {
			return EntityRef{}, false
		}
	}

	return entity, true
}

func (s *Storage) CheckChanged(tick Tick, query *Query, types []*ComponentType) {
	for _, ty := range types {
		if !ty.Comparable {
			continue
		}

		for _, archetype := range s.archetypes.All() {
			if !archetype.ContainsType(ty) {
				continue
			}

			if query != nil && !query.MatchesArchetype(archetype) {
				// the query did not return any values from this archetype,
				// so no way anything has changed
				continue
			}

			archetype.CheckChanged(tick, ty)
		}
	}
}

func (s *Storage) OptimizeQuery(query Query) *CachedQuery {
	return s.queryCache.Add(query)
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

// IterCachedQuery returns an iterator over entity refs matching the given query.
func (s *Storage) IterCachedQuery(q *CachedQuery, ctx QueryContext) QueryIter {
	return QueryIter{
		ctx:         ctx,
		query:       *q,
		accessorIdx: -1,
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
	ctx   QueryContext
	query CachedQuery

	row Row

	accessorIdx int
	entities    []EntityId
}

func (it *QueryIter) Next() (EntityRef, bool) {
	for {
		for int(it.row) < len(it.entities) {
			acc := &it.query.Accessors[it.accessorIdx]

			entity := EntityRef{
				fetch:     asFastSlice(acc.Columns),
				archetype: acc.Archetype,
				row:       it.row,
			}

			it.row += 1

			if it.query.IsArchetypeOnly || it.query.Matches(it.ctx, entity) {
				return entity, true
			}
		}

		// go to the next accessor
		it.accessorIdx += 1
		if it.accessorIdx >= len(it.query.Accessors) {
			break
		}

		// reset iterator
		it.row = 0
		it.entities = it.query.Accessors[it.accessorIdx].Archetype.entities
	}

	return EntityRef{}, false
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
