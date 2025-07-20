package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/oliverbestmann/byke/internal/set"
	"slices"
)

type Filter interface {
	applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter
}

type EmbeddableFilter interface {
	Filter
	embeddable(isEmbeddableMarker)
}

type FromEntityRef interface {
	fromEntityRef(ref arch.EntityRef)
}

type isEmbeddableMarker struct{}

type Option[C arch.IsComponent[C]] struct {
	value *C
}

func (Option[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	idx := result.Query.FetchComponent(arch.ComponentTypeOf[C](), true)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return []arch.Filter{
		{
			FetchOptional: []*arch.ComponentType{arch.ComponentTypeOf[C]()},
		},
	}
}

func (c *Option[C]) Get() (C, bool) {
	return c.OrZero(), c.value != nil
}

func (c *Option[C]) MustGet() C {
	return *c.value
}

func (c *Option[C]) OrZero() C {
	var zeroValue C
	return c.Or(zeroValue)
}

func (c *Option[C]) Or(fallbackValue C) C {
	if c.value != nil {
		return *c.value
	}

	return fallbackValue
}

type OptionMut[C arch.IsComponent[C]] struct {
	value *C
}

func (OptionMut[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()
	result.Mutable = append(result.Mutable, componentType)

	idx := result.Query.FetchComponent(arch.ComponentTypeOf[C](), true)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return []arch.Filter{
		{
			FetchOptional: []*arch.ComponentType{componentType},
		},
	}
}

func (c *OptionMut[C]) Get() (*C, bool) {
	return c.value, c.value != nil
}

func (c *OptionMut[C]) MustGet() *C {
	if c.value == nil {
		panic(fmt.Sprintf("%T is empty", *c))
	}

	return c.value
}

type Has[C arch.IsComponent[C]] struct {
	ptr uintptr
}

func (Has[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	idx := result.Query.FetchComponent(componentType, true)
	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return nil
}

func (h Has[C]) Exists() bool {
	return h.ptr != 0
}

type With[C arch.IsComponent[C]] struct{}

func (With[C]) embeddable(isEmbeddableMarker) {}

func (With[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			With: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				return entity.Get(componentType) != nil
			},
		},
	}
}

type Without[C arch.IsComponent[C]] struct{}

func (Without[C]) embeddable(isEmbeddableMarker) {}

func (Without[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			Without: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				return entity.Get(componentType) == nil
			},
		},
	}
}

type Changed[C arch.IsSupportsChangeDetectionComponent[C]] struct{}

func (Changed[C]) embeddable(isEmbeddableMarker) {}

func (Changed[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			With: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				tick := entity.Changed(componentType)
				return tick != arch.NoTick && tick >= q.LastRun
			},
		},
	}
}

type Added[C arch.IsComponent[C]] struct{}

func (Added[C]) embeddable(isEmbeddableMarker) {}

func (Added[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			With: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				tick := entity.Added(componentType)
				return tick != arch.NoTick && tick >= q.LastRun
			},
		},
	}
}

type Or[A, B Filter] struct{}

func (Or[A, B]) embeddable(isEmbeddableMarker) {}

func (Or[A, B]) applyTo(result *ParsedQuery, fieldOffset uintptr) []arch.Filter {
	var aZero A
	filterA := aZero.applyTo(result, 0)

	var bZero B
	filterB := bZero.applyTo(result, 0)

	// for And we can optimize: we can just move the intersection of
	// the With & Without types to the top filter

	// first collect with/without values of filter A
	var withA, withoutA set.Set[*arch.ComponentType]

	for _, filter := range filterA {
		withA.InsertAll(slices.Values(filter.With))
		withoutA.InsertAll(slices.Values(filter.Without))
	}

	// and then keep only the ones from B that are also in A
	var with, without []*arch.ComponentType

	for _, filter := range filterB {
		for _, ty := range filter.With {
			if withA.Has(ty) {
				with = append(with, ty)
			}
		}

		for _, ty := range filter.Without {
			if withoutA.Has(ty) {
				without = append(without, ty)
			}
		}
	}

	return []arch.Filter{
		{
			With:    with,
			Without: without,

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				return matches(filterA, q, entity) || matches(filterB, q, entity)
			},
		},
	}
}

func matches(filters []arch.Filter, q *arch.Query, entity arch.EntityRef) bool {
	for _, filter := range filters {
		if filter.Matches != nil && !filter.Matches(q, entity) {
			return false
		}
	}

	return true
}
