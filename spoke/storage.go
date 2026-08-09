package spoke

import (
	"fmt"
	"iter"
)

type Storage struct {
	entityToArchetype map[EntityId]*Archetype
	archetypes        ArchetypeGraph
	queryCache        queryCache

	// optional hooks for each component type
	hooks map[ComponentTypeId]ComponentHooks
}

func NewStorage() *Storage {
	storage := &Storage{
		entityToArchetype: map[EntityId]*Archetype{},
		hooks:             map[ComponentTypeId]ComponentHooks{},
	}

	storage.queryCache.archetypes = &storage.archetypes

	return storage
}

func (s *Storage) Spawn(tick Tick, entityId EntityId, components []ErasedComponent) {
	if _, exists := s.entityToArchetype[entityId]; exists {
		panic(fmt.Sprintf("entity %s already exists", entityId))
	}

	// collect the component types
	var componentTypes []*ComponentType
	for _, component := range components {
		componentTypes = append(componentTypes, component.ComponentType())
	}

	// find or create the archetype we fit into
	archetype, created := s.archetypes.Lookup(componentTypes)
	if created {
		s.handleNewArchetype(archetype)
	}

	// add entity to the archetype
	archetype.Insert(tick, entityId, components)

	// remember where we put the entity
	s.entityToArchetype[entityId] = archetype

	// call hooks
	s.dispatchOnAdd(archetype, entityId, componentTypes)
	s.dispatchOnInsert(archetype, entityId, componentTypes)
}

func (s *Storage) Despawn(entityId EntityId) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return false
	}

	// call hooks before removing the entity
	s.dispatchOnDiscard(archetype, entityId, archetype.Types)
	s.dispatchOnRemove(archetype, entityId, archetype.Types)
	s.dispatchOnDespawn(archetype, entityId, archetype.Types)

	archetype.Remove(entityId)

	delete(s.entityToArchetype, entityId)

	if archetype.Len() == 0 {
		// TODO maybe remove the archetype from the graph if it is empty?
		//  that might speed up queries matching a lot of empty archetypes
		//  we would also need to call the optimizer again
	}

	return true
}

func (s *Storage) InsertComponents(tick Tick, entityId EntityId, components []ErasedComponent) {
	prevArchetype, ok := s.entityToArchetype[entityId]
	if !ok {
		panic(fmt.Sprintf("entity %s does not exist", entityId))
	}

	newArchetype := prevArchetype

	var addedComponentTypes []*ComponentType
	var updatedComponentTypes []*ComponentType

	var created, anyCreated bool
	for _, component := range components {
		componentType := component.ComponentType()
		if newArchetype.ContainsType(componentType) {
			updatedComponentTypes = append(updatedComponentTypes, componentType)
			continue
		}

		// we need to move to a new archetype
		newArchetype, created = s.archetypes.NextWith(newArchetype, componentType)
		anyCreated = anyCreated || created

		// remember added types to call the correct hooks later
		addedComponentTypes = append(addedComponentTypes, componentType)
	}

	if anyCreated {
		s.handleNewArchetype(newArchetype)
	}

	// before updating any values, we need to discard the previous values,
	// but only for the types that are being updated
	s.dispatchOnDiscard(prevArchetype, entityId, updatedComponentTypes)

	if newArchetype == prevArchetype {
		// no change in archetypes, update in existing archetype
		for idx, component := range components {
			components[idx] = prevArchetype.ReplaceComponentValue(tick, entityId, component)
		}

		// call insert hook after updating the entity
		s.dispatchOnInsert(newArchetype, entityId, updatedComponentTypes)

		return
	}

	// transfer our entity
	newArchetype.Import(tick, prevArchetype, entityId, components...)

	// remove from the previous archetype
	prevArchetype.Remove(entityId)

	// and update the index
	s.entityToArchetype[entityId] = newArchetype

	// after updating the existing components, we need to trigger their insert hook
	s.dispatchOnInsert(newArchetype, entityId, updatedComponentTypes)

	// now also call hooks for added components
	s.dispatchOnAdd(newArchetype, entityId, addedComponentTypes)
	s.dispatchOnInsert(newArchetype, entityId, addedComponentTypes)

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
		// call hook to discard the previous value
		s.dispatchOnDiscard(archetype, entityId, []*ComponentType{componentType})

		// update the value of the component
		componentValue := archetype.ReplaceComponentValue(tick, entityId, component)

		// call insert hook after updating the entity
		s.dispatchOnInsert(archetype, entityId, []*ComponentType{componentType})

		return componentValue
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

	// the component was freshly added to this entity
	s.dispatchOnAdd(newArchetype, entityId, []*ComponentType{componentType})
	s.dispatchOnInsert(newArchetype, entityId, []*ComponentType{componentType})

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

	// call hooks before removing the component
	s.dispatchOnDiscard(archetype, entityId, []*ComponentType{componentType})
	s.dispatchOnRemove(archetype, entityId, []*ComponentType{componentType})

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

func (s *Storage) Get(entityId EntityId) (EntityRef, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return EntityRef{}, false
	}

	return archetype.Get(entityId)
}

func (s *Storage) GetWithQuery(q *CachedQuery, qc QueryContext, entityId EntityId) (EntityRef, bool) {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		return EntityRef{}, false
	}

	if !q.MatchesArchetypeWithQueryContext(qc, archetype) {
		return EntityRef{}, false
	}

	accessorIdx, ok := q.Archetypes[archetype.Id]
	if !ok {
		return EntityRef{}, false
	}

	entity, ok := archetype.Get(entityId)
	if !ok {
		panic("archetype does not contain entity")
	}

	if !q.IsArchetypeOnly && !q.Matches(qc, entity) {
		return EntityRef{}, false
	}

	entity.fetch = unsafeSlice(q.Accessors[accessorIdx].Columns)

	return entity, true
}

func (s *Storage) CheckChanged(tick Tick, query *CachedQuery, types []*ComponentType) {
	for _, ty := range types {
		if !ty.DirtyTracking {
			continue
		}

		for idx := range query.Accessors {
			ac := &query.Accessors[idx]

			if !ac.Archetype.ContainsType(ty) {
				continue
			}

			if !query.MatchesArchetype(ac.Archetype) {
				// the query did not return any values from this archetype,
				// so no way anything has changed
				continue
			}

			ac.Archetype.CheckChanged(tick, ty)
		}
	}
}

func (s *Storage) OptimizeQuery(query Query) *CachedQuery {
	return s.queryCache.Add(query)
}

func (s *Storage) handleNewArchetype(newArchetype *Archetype) {
	doOptimize := func() { s.queryCache.Optimize(newArchetype) }

	// a new archetype was created,
	// we might need to re-optimize some queries
	doOptimize()

	// we register a callback to re-optimize all queries that are looking at data
	// of one of the columns to update any changed pointers
	for _, column := range newArchetype.columns {
		column.OnGrow(doOptimize)
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
func (s *Storage) IterQuery(q *CachedQuery, ctx QueryContext) QueryIter {
	return QueryIter{
		qc:          ctx,
		query:       *q,
		accessorIdx: -1,
	}
}

func (s *Storage) HasComponent(entityId EntityId, componentType *ComponentType) bool {
	archetype, ok := s.entityToArchetype[entityId]
	if !ok {
		// the entity itself does not exist
		return false
	}

	return archetype.ContainsType(componentType)
}

func (s *Storage) EntityCount() int {
	return len(s.entityToArchetype)
}

func (s *Storage) RegisterComponentHooks(componentType *ComponentType) *RegisterComponentHooks {
	return &RegisterComponentHooks{
		storage:       s,
		componentType: componentType,
		hooks:         s.hooks[componentType.Id],
	}
}

func (s *Storage) dispatchOnAdd(archetype *Archetype, entityId EntityId, components []*ComponentType) {
	for _, ty := range components {
		if hook := s.hooks[ty.Id].OnAdd; hook != nil {
			hook(archetype.mustGet(entityId), ty)
		}
	}
}

func (s *Storage) dispatchOnInsert(archetype *Archetype, entityId EntityId, components []*ComponentType) {
	for _, ty := range components {
		if hook := s.hooks[ty.Id].OnInsert; hook != nil {
			hook(archetype.mustGet(entityId), ty)
		}
	}
}

func (s *Storage) dispatchOnDiscard(archetype *Archetype, entityId EntityId, components []*ComponentType) {
	for _, ty := range components {
		if hook := s.hooks[ty.Id].OnDiscard; hook != nil {
			hook(archetype.mustGet(entityId), ty)
		}
	}
}

func (s *Storage) dispatchOnRemove(archetype *Archetype, entityId EntityId, components []*ComponentType) {
	for _, ty := range components {
		if hook := s.hooks[ty.Id].OnRemove; hook != nil {
			hook(archetype.mustGet(entityId), ty)
		}
	}
}

func (s *Storage) dispatchOnDespawn(archetype *Archetype, entityId EntityId, components []*ComponentType) {
	for _, ty := range components {
		if hook := s.hooks[ty.Id].OnDespawn; hook != nil {
			hook(archetype.mustGet(entityId), ty)
		}
	}
}

type QueryIter struct {
	qc    QueryContext
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
				fetch:     unsafeSlice(acc.Columns),
				archetype: acc.Archetype,
				row:       it.row,
			}

			it.row += 1

			if it.query.IsArchetypeOnly || it.query.Matches(it.qc, entity) {
				return entity, true
			}
		}

		// go to the next accessor
		it.accessorIdx += 1
		if it.accessorIdx >= len(it.query.Accessors) {
			break
		}

		ac := &it.query.Accessors[it.accessorIdx]

		if !it.query.MatchesArchetypeWithQueryContext(it.qc, ac.Archetype) {
			continue
		}

		// reset iterator
		it.row = 0
		it.entities = ac.Archetype.entities
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
