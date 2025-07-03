package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
)

type Filter interface {
	FromEntityRef
	applyTo(result *ParsedQuery)
}

type isEmbeddableMarker struct{}

type EmbeddableFilter interface {
	embeddable(isEmbeddableMarker)
}

type FromEntityRef interface {
	fromEntityRef(ref arch.EntityRef)
}

type Option[C arch.IsComponent[C]] struct {
	value *C
}

func (*Option[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchOptional = append(result.Query.FetchOptional, componentType)
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

func (*OptionMut[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchOptional = append(result.Query.FetchOptional, componentType)
	result.Mutable = append(result.Mutable, componentType)
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

func (*Has[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchHas = append(result.Query.FetchHas, componentType)
}

func (c *Has[C]) fromEntityRef(ref arch.EntityRef) {
	_, ok := ref.Get(arch.ComponentTypeOf[C]())
	c.Exists = ok
}

type With[C arch.IsComponent[C]] struct{}

func (With[C]) embeddable(isEmbeddableMarker) {}

func (*With[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.With = append(result.Query.With, componentType)
}

func (*With[C]) fromEntityRef(ref arch.EntityRef) {
	// TODO maybe get rid of this
	// does not need to do anything
}

type Without[C arch.IsComponent[C]] struct{}

func (Without[C]) embeddable(isEmbeddableMarker) {}

func (*Without[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.Without = append(result.Query.Without, componentType)
}

func (*Without[C]) fromEntityRef(ref arch.EntityRef) {
	// TODO maybe get rid of this
	// does not need to do anything
}

type Changed[C arch.IsComparableComponent[C]] struct{}

func (Changed[C]) embeddable(isEmbeddableMarker) {}

func (*Changed[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.WithChanged = append(result.Query.WithChanged, componentType)
}

func (*Changed[C]) fromEntityRef(ref arch.EntityRef) {
	// TODO maybe get rid of this
	// does not need to do anything
}

type Added[C arch.IsComparableComponent[C]] struct{}

func (Added[C]) embeddable(isEmbeddableMarker) {}

func (*Added[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.WithAdded = append(result.Query.WithAdded, componentType)
}

func (*Added[C]) fromEntityRef(ref arch.EntityRef) {
	// TODO maybe get rid of this
	// does not need to do anything
}
