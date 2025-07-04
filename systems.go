package byke

import (
	"github.com/oliverbestmann/byke/internal/set"
	"reflect"
)

type SystemId uint64

type AnySystem any

type AsSystemConfigs interface {
	AsSystemConfigs() []SystemConfig
}

func asSystemConfig(value AnySystem) SystemConfig {
	switch value := value.(type) {
	case SystemConfig:
		return value

	default:
		return SystemConfig{
			Id: systemIdOf(value),
			fn: reflect.ValueOf(value),
		}
	}
}

func asSystemConfigs(values ...AnySystem) []SystemConfig {
	var configs []SystemConfig

	for _, value := range values {
		switch value := value.(type) {
		case []SystemConfig:
			configs = append(configs, value...)

		case AsSystemConfigs:
			configs = append(configs, value.AsSystemConfigs()...)

		default:
			configs = append(configs, asSystemConfig(value))
		}
	}

	return configs
}

func mergeConfigs(configs []SystemConfig) {
	merged := map[SystemId]SystemConfig{}

	for _, config := range configs {
		existing, ok := merged[config.Id]
		if !ok {
			config = config.MergeWith(existing)
		}

		merged[config.Id] = config
	}
}

func System(systems ...AnySystem) Systems {
	return Systems{
		systems: systems,
	}
}

func systemIdOf(system any) SystemId {
	fn := reflect.ValueOf(system)
	if fn.Kind() != reflect.Func {
		panic("system is not a function")
	}

	return SystemId(uintptr(fn.UnsafePointer()))
}

func SystemChain(systems ...AnySystem) AnySystem {
	allSystems := asSystemConfigs(systems...)

	for idx := 0; idx < len(allSystems)-1; idx++ {
		allSystems[idx].before.Insert(allSystems[idx+1].Id)
	}

	return allSystems
}

type SystemConfig struct {
	Id SystemId

	// the actual fn, must be a function
	fn         reflect.Value
	before     set.Set[SystemId]
	after      set.Set[SystemId]
	predicates []func() bool
}

func (conf SystemConfig) MergeWith(other SystemConfig) SystemConfig {
	if conf.Id != other.Id {
		panic("can not merge systems with different ids")
	}

	//goland:noinspection GoShadowedVar
	copy := conf

	copy.before = set.FromValues(copy.before.Values())
	copy.before.InsertAll(other.before.Values())

	copy.after = set.FromValues(copy.after.Values())
	copy.after.InsertAll(other.after.Values())

	copy.predicates = append(append([]func() bool{}, copy.predicates...), other.predicates...)

	return copy
}

type Systems struct {
	systems []AnySystem

	after      set.Set[SystemId]
	before     set.Set[SystemId]
	predicates []func() bool
}

func (s Systems) AsSystemConfigs() []SystemConfig {
	systems := asSystemConfigs(s.systems...)

	for idx := range systems {
		system := &systems[idx]
		system.after.InsertAll(s.after.Values())
		system.before.InsertAll(s.before.Values())
		system.predicates = append(system.predicates, s.predicates...)
	}

	return systems
}

func (s Systems) After(other AnySystem) Systems {
	for _, system := range asSystemConfigs(other) {
		s.after.Insert(system.Id)
	}

	return s
}

func (s Systems) Before(other AnySystem) Systems {
	for _, system := range asSystemConfigs(other) {
		s.before.Insert(system.Id)
	}

	return s
}

func (s Systems) RunIf(predicate func() bool) Systems {
	s.predicates = append(s.predicates, predicate)
	return s
}
