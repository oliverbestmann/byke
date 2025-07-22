package spoke

import (
	"weak"
)

type CachedQuery struct {
	Query
	Accessors  []archetypeWithColumns
	Archetypes map[ArchetypeId]int
}

type archetypeWithColumns struct {
	Archetype *Archetype
	Columns   []ColumnAccess
}

type queryCache struct {
	archetypes *ArchetypeGraph
	queries    []weak.Pointer[CachedQuery]
}

func (qc *queryCache) Add(query Query) *CachedQuery {
	cached := &CachedQuery{
		Query: query,
	}

	qc.queries = append(qc.queries, weak.Make(cached))
	qc.optimizeQuery(cached)

	return cached
}

func (qc *queryCache) Optimize(newArchetype *Archetype) {
	// reuse slice memory
	alive := qc.queries[:0]

	for _, weakQuery := range qc.queries {
		query := weakQuery.Value()
		if query == nil {
			continue
		}

		alive = append(alive, weakQuery)

		if newArchetype == nil || query.MatchesArchetype(newArchetype) {
			qc.optimizeQuery(query)
		}
	}

	qc.queries = alive
}

func (qc *queryCache) optimizeQuery(query *CachedQuery) {
	query.Accessors = query.Accessors[:0]

	clear(query.Archetypes)

	if query.Archetypes == nil {
		query.Archetypes = map[ArchetypeId]int{}
	}

	for _, archetype := range qc.archetypes.All() {
		if !query.MatchesArchetype(archetype) {
			continue
		}

		_, columns := archetype.IterForQuery(&query.Query, nil)

		query.Archetypes[archetype.Id] = len(query.Accessors)

		query.Accessors = append(query.Accessors, archetypeWithColumns{
			Archetype: archetype,
			Columns:   columns,
		})
	}
}
