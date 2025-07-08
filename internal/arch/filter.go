package arch

type Filter struct {
	// The archetype needs to have all of those component types.
	With []*ComponentType

	// The archetype must not have those component types
	Without []*ComponentType

	// Fetch if possible
	FetchOptional []*ComponentType

	// Check if a entity matches this filter.
	Matches func(q *Query, entity EntityRef) bool
}

type Query struct {
	// tick that the system was last run.
	// This is used to filter for changed or added components.
	LastRun Tick

	// components we want to actually read
	Fetch []*ComponentType

	// components we just want to check if they exist
	FetchHas []*ComponentType

	// more general filters, such as nested Or or And
	Filters []Filter
}

func (q *Query) MatchesArchetype(a *Archetype) bool {
	if !containsAllTypes(a, q.Fetch) {
		return false
	}

	for _, filter := range q.Filters {
		// must contain all types from With
		if !containsAllTypes(a, filter.With) {
			return false
		}

		// negative check for Without
		for _, ty := range filter.Without {
			if a.ContainsType(ty) {
				return false
			}
		}
	}

	return true
}

// Matches must only be run for entities provided by an Archetype that matched MatchesArchetype.
func (q *Query) Matches(entity EntityRef) bool {
	// apply filters
	for _, filter := range q.Filters {
		if filter.Matches != nil && !filter.Matches(q, entity) {
			return false
		}
	}

	return true
}

func containsAllTypes(a *Archetype, types []*ComponentType) bool {
	for _, ty := range types {
		if !a.ContainsType(ty) {
			return false
		}
	}

	return true
}
