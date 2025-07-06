package byke

// SystemSet groups multiple systems together within one ScheduleId.
type SystemSet struct {
	after      []*SystemSet
	before     []*SystemSet
	predicates []AnySystem
}

func (s *SystemSet) After(other *SystemSet) *SystemSet {
	s.after = append(s.after, other)
	return s
}

func (s *SystemSet) Before(other *SystemSet) *SystemSet {
	s.before = append(s.before, other)
	return s
}

func (s *SystemSet) RunIf(predicate AnySystem) *SystemSet {
	s.predicates = append(s.predicates, predicate)
	return s
}
