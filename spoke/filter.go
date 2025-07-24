package spoke

type FetchComponent struct {
	ComponentType *ComponentType
	Optional      bool
}

type QueryBuilder struct {
	Fetch   []FetchComponent
	Filters []Filter
}

func (q *QueryBuilder) Filter(f Filter) {
	if !f.IsZero() {
		q.Filters = append(q.Filters, f)
	}
}

func (q *QueryBuilder) FetchComponent(componentType *ComponentType, optional bool) int {
	for idx := range q.Fetch {
		fetch := &q.Fetch[idx]
		if fetch.ComponentType == componentType {
			fetch.Optional = fetch.Optional && optional
			return idx
		}
	}

	idx := len(q.Fetch)

	q.Fetch = append(q.Fetch, FetchComponent{
		ComponentType: componentType,
		Optional:      optional,
	})

	return idx
}

func (q *QueryBuilder) IsArchetypeOnly() bool {
	for idx := range q.Filters {
		if !q.Filters[idx].IsArchetypeOnly() {
			return false
		}
	}

	return true
}

func (q *QueryBuilder) Build() Query {
	return Query{
		Fetch:           q.Fetch,
		Filters:         q.Filters,
		IsArchetypeOnly: q.IsArchetypeOnly(),
	}
}

type Query struct {
	// components we want to actually read
	Fetch []FetchComponent

	// more general Filters, such as nested Or or And
	Filters []Filter

	// true if this is an archetype only query
	IsArchetypeOnly bool
}

func (q *Query) MatchesArchetype(a *Archetype) bool {
	for _, ty := range q.Fetch {
		if !a.ContainsType(ty.ComponentType) && !ty.Optional {
			return false
		}
	}

	for idx := range q.Filters {
		if !q.Filters[idx].MatchesArchetype(a) {
			return false
		}
	}

	return true
}

// MatchesArchetypeWithQueryContext must only be called if MatchesArchetype already returns true.
func (q *Query) MatchesArchetypeWithQueryContext(qc QueryContext, a *Archetype) bool {
	if q.IsArchetypeOnly {
		return true
	}

	for idx := range q.Filters {
		if !q.Filters[idx].MatchesArchetypeWithQueryContext(qc, a) {
			return false
		}
	}

	return true
}

// Matches must only be run for entities provided by an Archetype that matched MatchesArchetype.
// If the query IsArchetypeOnly, this method does not need to be called.
func (q *Query) Matches(ctx QueryContext, entity EntityRef) bool {
	for idx := range q.Filters {
		if !q.Filters[idx].Matches(ctx, entity) {
			return false
		}
	}

	return true
}

type Filter struct {
	// The archetype needs to have this component type
	With *ComponentType

	// The archetype must not have this component type
	Without *ComponentType

	// The archetype must have this component newly added
	Added *ComponentType

	// The archetype must have this component changed
	Changed *ComponentType

	// More Filters, each combined with an or.
	Or []Filter
}

func (f *Filter) IsZero() bool {
	for _, or := range f.Or {
		if !or.IsZero() {
			return false
		}
	}

	return f.With == nil && f.Without == nil && f.Added == nil && f.Changed == nil
}

func (f *Filter) IsArchetypeOnly() bool {
	if f.Added != nil || f.Changed != nil {
		return false
	}

	for idx := range f.Or {
		if !f.Or[idx].IsArchetypeOnly() {
			return false
		}
	}

	return true
}

func (f *Filter) MatchesArchetype(a *Archetype) bool {
	if ty := f.With; ty != nil && !a.ContainsType(ty) {
		return false
	}

	if ty := f.Without; ty != nil && a.ContainsType(ty) {
		return false
	}

	if ty := f.Added; ty != nil && !a.ContainsType(ty) {
		return false
	}

	if ty := f.Changed; ty != nil && !a.ContainsType(ty) {
		return false
	}

	if len(f.Or) == 0 {
		return true
	}

	for idx := range f.Or {
		if f.Or[idx].MatchesArchetype(a) {
			return true
		}
	}

	return false
}

// MatchesArchetypeWithQueryContext checks for QueryContext dependent checks on the
// archetype itself. This must only be called if MatchesArchetype already returns true.
func (f *Filter) MatchesArchetypeWithQueryContext(qc QueryContext, a *Archetype) bool {
	if f.Added != nil {
		tick := a.LastAdded(f.Added)
		if tick == NoTick || tick < qc.LastRun {
			return false
		}
	}

	if f.Changed != nil {
		tick := a.LastChanged(f.Changed)
		if tick == NoTick || tick < qc.LastRun {
			return false
		}
	}

	if len(f.Or) == 0 {
		return true
	}

	for _, filter := range f.Or {
		if filter.MatchesArchetypeWithQueryContext(qc, a) {
			return true
		}
	}

	return false
}

// Matches checks the non-archetype specific checks. This must only be called for
// an entity from an Archetype accepted by MatchesArchetype.
func (f *Filter) Matches(qc QueryContext, entity EntityRef) bool {
	if f.Added != nil {
		tick := entity.Added(f.Added)
		if tick == NoTick || tick < qc.LastRun {
			return false
		}
	}

	if f.Changed != nil {
		tick := entity.Changed(f.Changed)
		if tick == NoTick || tick < qc.LastRun {
			return false
		}
	}

	if len(f.Or) == 0 {
		return true
	}
	for _, filter := range f.Or {
		if filter.Matches(qc, entity) {
			return true
		}
	}

	return false
}

type QueryContext struct {
	// Last time that the system running this query was executed
	LastRun Tick
}
