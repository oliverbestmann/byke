package query

import "github.com/oliverbestmann/byke/internal/arch"

type Filter interface {
	applyTo(result *ParsedQuery)
}

type Option[C arch.IsComponent[C]] struct {
	value *C
}

func (Option[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchOptional = append(result.Query.FetchOptional, componentType)
}

type OptionMut[C arch.IsComponent[C]] struct {
	value *C
}

func (OptionMut[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchOptional = append(result.Query.FetchOptional, componentType)
	result.Mutable = append(result.Mutable, componentType)
}

type Has[C arch.IsComponent[C]] struct {
	Exists bool
}

func (Has[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.FetchHas = append(result.Query.FetchHas, componentType)
}

type With[C arch.IsComponent[C]] struct{}

func (With[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.With = append(result.Query.With, componentType)
}

type Without[C arch.IsComponent[C]] struct{}

func (Without[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.Without = append(result.Query.Without, componentType)
}

type Changed[C arch.IsComparableComponent[C]] struct{}

func (Changed[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.WithChanged = append(result.Query.WithChanged, componentType)
}

type Added[C arch.IsComparableComponent[C]] struct{}

func (Added[C]) applyTo(result *ParsedQuery) {
	componentType := arch.ComponentTypeOf[C]()
	result.Query.WithAdded = append(result.Query.WithAdded, componentType)
}
