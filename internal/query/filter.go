package query

import (
	"fmt"
	spoke2 "github.com/oliverbestmann/byke/spoke"
)

type Filter interface {
	applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter
}

type EmbeddableFilter interface {
	Filter
	embeddable(isEmbeddableMarker)
}

type isEmbeddableMarker struct{}

type Option[C spoke2.IsComponent[C]] struct {
	value *C
}

func (Option[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	idx := result.Builder.FetchComponent(spoke2.ComponentTypeOf[C](), true)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke2.Filter{}
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

type OptionMut[C spoke2.IsComponent[C]] struct {
	value *C
}

func (OptionMut[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	componentType := spoke2.ComponentTypeOf[C]()
	result.Mutable = append(result.Mutable, componentType)

	idx := result.Builder.FetchComponent(spoke2.ComponentTypeOf[C](), true)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke2.Filter{}
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

type Has[C spoke2.IsComponent[C]] struct {
	ptr uintptr
}

func (h Has[C]) Exists() bool {
	return h.ptr != 0
}

func (Has[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	componentType := spoke2.ComponentTypeOf[C]()

	idx := result.Builder.FetchComponent(componentType, true)
	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke2.Filter{}
}

type With[C spoke2.IsComponent[C]] struct{}

func (With[C]) embeddable(isEmbeddableMarker) {}

func (With[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	return spoke2.Filter{
		With: spoke2.ComponentTypeOf[C](),
	}
}

type Without[C spoke2.IsComponent[C]] struct{}

func (Without[C]) embeddable(isEmbeddableMarker) {}

func (Without[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	return spoke2.Filter{
		Without: spoke2.ComponentTypeOf[C](),
	}
}

type Changed[C spoke2.IsSupportsChangeDetectionComponent[C]] struct{}

func (Changed[C]) embeddable(isEmbeddableMarker) {}

func (Changed[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	return spoke2.Filter{
		Changed: spoke2.ComponentTypeOf[C](),
	}
}

type Added[C spoke2.IsComponent[C]] struct{}

func (Added[C]) embeddable(isEmbeddableMarker) {}

func (Added[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	return spoke2.Filter{
		Added: spoke2.ComponentTypeOf[C](),
	}
}

type Or[A, B Filter] struct{}

func (Or[A, B]) embeddable(isEmbeddableMarker) {}

func (Or[A, B]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke2.Filter {
	var aZero A
	filterA := aZero.applyTo(result, fieldOffset)

	var bZero B
	filterB := bZero.applyTo(result, fieldOffset)

	return spoke2.Filter{
		Or: []spoke2.Filter{
			filterA,
			filterB,
		},
	}
}
