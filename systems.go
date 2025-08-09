package byke

import (
	"fmt"
	"reflect"
	"slices"
	"unsafe"

	"github.com/oliverbestmann/byke/internal/set"
)

// SystemId uniquely identifies a system.
//
// Implementation note: this currently uses the function pointer. A function pointer in go
// has two layers of indirection to be able to handle closures correctly. The function pointer points
// to a small memory region holding a pointer to the actual function, as well as all data closed
// over by a closure. The SystemId currently points to this outer region of memory,
// not to the function code itself.
type SystemId unsafe.Pointer

// AnySystem can be anything that can be coerced into a system.
type AnySystem any

func asSystemConfig(value AnySystem) *systemConfig {
	switch value := value.(type) {
	case *systemConfig:
		return value

	default:
		return &systemConfig{
			Id:         systemIdOf(value),
			SystemFunc: reflect.ValueOf(value),
		}
	}
}

func asSystemConfigs(values ...AnySystem) []*systemConfig {
	var configs []*systemConfig

	for _, value := range values {
		switch value := value.(type) {
		case []*systemConfig:
			configs = append(configs, value...)

		case Systems:
			configs = append(configs, value.asSystemConfigs()...)

		default:
			configs = append(configs, asSystemConfig(value))
		}
	}

	return mergeConfigs(configs)
}

func mergeConfigs(configs []*systemConfig) []*systemConfig {
	if len(configs) == 1 {
		// no need to merge, we just have one config
		return configs
	}

	// we use a slice here to ensure we keep any ordering
	var merged []*systemConfig

	for _, config := range configs {
		if slices.Contains(merged, config) {
			// pointer already in merged
			continue
		}

		// check the existing configs first
		idx := slices.IndexFunc(merged, func(c *systemConfig) bool { return c.Id == config.Id })

		if idx == -1 {
			merged = append(merged, config)
			continue
		}

		merged[idx].MergeWith(config)
	}

	return merged
}

func systemIdOf(systemFunc any) SystemId {
	fn := reflect.ValueOf(systemFunc)
	if fn.Kind() != reflect.Func {
		panic(fmt.Sprintf("system is not a function: %T", systemFunc))
	}

	// get the pointer to the funcval and take that one as the systems Id
	type eface struct{ typ, val unsafe.Pointer }
	funcval := (*eface)(unsafe.Pointer(&systemFunc)).val

	return SystemId(funcval)
}

type systemConfig struct {
	Id SystemId

	// the actual function
	SystemFunc reflect.Value

	Before     set.Set[SystemId]
	After      set.Set[SystemId]
	SystemSets set.Set[*SystemSet]

	Predicates []AnySystem
}

func (conf *systemConfig) MergeWith(other *systemConfig) *systemConfig {
	if conf.Id != other.Id {
		panic("can not merge systems with different ids")
	}

	conf.Before.InsertAll(other.Before.Values())
	conf.After.InsertAll(other.After.Values())
	conf.SystemSets.InsertAll(other.SystemSets.Values())
	conf.Predicates = append(conf.Predicates, other.Predicates...)

	return conf
}

// Systems wrap one or more systems. See System for more details.
type Systems struct {
	systems []AnySystem

	after  set.Set[SystemId]
	before set.Set[SystemId]
	sets   set.Set[*SystemSet]

	predicates []AnySystem
}

// System wraps one or multiple systems (raw functions, other Systems) as a Systems instance.
// The Systems instance can then be used to configure inter system ordering or run predicates.
func System(systems ...AnySystem) Systems {
	return Systems{
		systems: systems,
	}
}

// After adds an ordering constraint to require all systems to run after the provided systems.
func (s Systems) After(other AnySystem) Systems {
	for _, system := range asSystemConfigs(other) {
		s.after.Insert(system.Id)
	}

	return s
}

// Before adds an ordering constraint to require all systems to run before the provided systems.
func (s Systems) Before(other AnySystem) Systems {
	for _, system := range asSystemConfigs(other) {
		s.before.Insert(system.Id)
	}

	return s
}

// InSet marks the systems to be part of the given SystemSet.
func (s Systems) InSet(systemSet *SystemSet) Systems {
	s.sets.Insert(systemSet)

	return s
}

// RunIf appends a run predicate to all systems. The predicate will be executed for
// each system in turn to determine if the system it should be executed or not.
func (s Systems) RunIf(predicate AnySystem) Systems {
	s.predicates = append(s.predicates, predicate)
	return s
}

// Chain chains the systems to run one after another.
func (s Systems) Chain() Systems {
	systems := s.asSystemConfigs()

	for idx := 0; idx < len(systems)-1; idx++ {
		systems[idx].Before.Insert(systems[idx+1].Id)
	}

	var anySystems []AnySystem
	for _, system := range systems {
		anySystems = append(anySystems, system)
	}

	return System(anySystems...)
}

func (s Systems) asSystemConfigs() []*systemConfig {
	systems := asSystemConfigs(s.systems...)

	for idx := range systems {
		system := systems[idx]
		system.After.InsertAll(s.after.Values())
		system.Before.InsertAll(s.before.Values())
		system.SystemSets.InsertAll(s.sets.Values())
		system.Predicates = append(system.Predicates, s.predicates...)
	}

	return systems
}
