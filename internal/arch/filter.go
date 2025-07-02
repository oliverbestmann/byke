package arch

type Query struct {
	// tick that the system was last run.
	// This is used to filter for changed or added components.
	LastRun uint64

	// components we want to actually read
	Fetch []*ComponentType

	// components to fetch if the entity has them
	FetchOptional []*ComponentType

	// components we just want to check if they exist
	FetchHas []*ComponentType

	// components that the entity must have but we do not necessarily want to fetch
	With        []*ComponentType
	WithAdded   []*ComponentType
	WithChanged []*ComponentType

	// components the entity must not have
	Without []*ComponentType
}

func (q *Query) MatchesArchetype(a *Archetype) bool {
	if !containsAllTypes(a, q.Fetch) {
		return false
	}

	if !containsAllTypes(a, q.With) {
		return false
	}

	if !containsAllTypes(a, q.WithAdded) {
		return false
	}

	if !containsAllTypes(a, q.WithChanged) {
		return false
	}

	// negative check for Without
	for _, ty := range q.Without {
		if a.ContainsType(ty) {
			return false
		}
	}

	return true
}

// Matches must only be run for entities provided by an Archetype that matched MatchesArchetype.
func (q *Query) Matches(entity EntityRef) bool {
	components := ComponentValues(entity.Components)

	for _, ty := range q.WithAdded {
		value, ok := components.ByType(ty)
		if !ok {
			// should not happen
			return false
		}

		if value.Added < q.LastRun {
			return false
		}
	}

	for _, ty := range q.WithChanged {
		value, ok := components.ByType(ty)
		if !ok {
			// should not happen
			return false
		}

		if value.Changed < q.LastRun {
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
