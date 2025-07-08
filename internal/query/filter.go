package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
)

type Filter interface {
	applyTo(result *ParsedQuery) []arch.Filter
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

func (Option[C]) applyTo(result *ParsedQuery) []arch.Filter {
	return []arch.Filter{
		{
			FetchOptional: []*arch.ComponentType{arch.ComponentTypeOf[C]()},
		},
	}
}

func (c *Option[C]) fromEntityRef(ref arch.EntityRef) {
	value, ok := ref.Get(arch.ComponentTypeOf[C]())
	if ok {
		c.value = any(value.Value).(*C)
	} else {
		c.value = nil
	}
}

func (c *Option[C]) Get() (C, bool) {
	return c.OrZero(), c.value != nil
}

func (c *Option[C]) MustGet() C {
	return *c.value
}

func (c *Option[C]) OrZero() C {
	if c.value != nil {
		return *c.value
	}

	var zeroValue C
	return zeroValue
}

type OptionMut[C arch.IsComponent[C]] struct {
	value *C
}

func (OptionMut[C]) applyTo(result *ParsedQuery) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()
	result.Mutable = append(result.Mutable, componentType)

	return []arch.Filter{
		{
			FetchOptional: []*arch.ComponentType{componentType},
		},
	}
}

func (c *OptionMut[C]) fromEntityRef(ref arch.EntityRef) {
	value, ok := ref.Get(arch.ComponentTypeOf[C]())
	if ok {
		c.value = any(value.Value).(*C)
	} else {
		c.value = nil
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
	Exists bool
}

func (Has[C]) applyTo(result *ParsedQuery) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchHas = append(result.Query.FetchHas, componentType)
	return nil
}

func (c *Has[C]) fromEntityRef(ref arch.EntityRef) {
	_, ok := ref.Get(arch.ComponentTypeOf[C]())
	c.Exists = ok
}

type With[C arch.IsComponent[C]] struct{}

func (With[C]) embeddable(isEmbeddableMarker) {}

func (With[C]) applyTo(result *ParsedQuery) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			With: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				_, ok := entity.Get(componentType)
				return ok
			},
		},
	}
}

type Without[C arch.IsComponent[C]] struct{}

func (Without[C]) embeddable(isEmbeddableMarker) {}

func (Without[C]) applyTo(result *ParsedQuery) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			Without: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				_, ok := entity.Get(componentType)
				return !ok
			},
		},
	}
}

type Changed[C arch.IsComparableComponent[C]] struct{}

func (Changed[C]) embeddable(isEmbeddableMarker) {}

func (Changed[C]) applyTo(result *ParsedQuery) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			With: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				value, ok := entity.Get(componentType)
				if !ok {
					return false
				}

				if value.Changed < q.LastRun {
					return false
				}

				return true
			},
		},
	}
}

type Added[C arch.IsComparableComponent[C]] struct{}

func (Added[C]) embeddable(isEmbeddableMarker) {}

func (Added[C]) applyTo(result *ParsedQuery) []arch.Filter {
	componentType := arch.ComponentTypeOf[C]()

	return []arch.Filter{
		{
			With: []*arch.ComponentType{componentType},

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				value, ok := entity.Get(componentType)
				if !ok {
					return false
				}

				if value.Added < q.LastRun {
					return false
				}

				return true
			},
		},
	}
}

type Or[A, B Filter] struct{}

func (Or[A, B]) embeddable(isEmbeddableMarker) {}

func (Or[A, B]) applyTo(result *ParsedQuery) []arch.Filter {
	var aZero A
	filterA := aZero.applyTo(result)

	var bZero B
	filterB := bZero.applyTo(result)

	// TODO optimize by pulling out the union of With and Without

	return []arch.Filter{
		{
			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				return matches(filterA, q, entity) || matches(filterB, q, entity)
			},
		},
	}
}

type And[A, B Filter] struct{}

func (And[A, B]) embeddable(isEmbeddableMarker) {}

func (And[A, B]) applyTo(result *ParsedQuery) []arch.Filter {
	var aZero A
	filterA := aZero.applyTo(result)

	var bZero B
	filterB := bZero.applyTo(result)

	// for and we can optimize. We can just move the With & Without types
	// to the top filter
	var with, without []*arch.ComponentType

	for _, filter := range filterA {
		with = append(with, filter.With...)
		without = append(without, filter.With...)
	}

	for _, filter := range filterB {
		with = append(with, filter.With...)
		without = append(without, filter.With...)
	}

	return []arch.Filter{
		{
			With:    with,
			Without: without,

			Matches: func(q *arch.Query, entity arch.EntityRef) bool {
				return matches(filterA, q, entity) && matches(filterB, q, entity)
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
