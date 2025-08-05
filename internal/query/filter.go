package query

import (
	"fmt"
	"reflect"

	"github.com/oliverbestmann/byke/internal/refl"
	spoke "github.com/oliverbestmann/byke/spoke"
)

type Filter interface {
	applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter
}

type EmbeddableFilter interface {
	Filter
	embeddable(isEmbeddableMarker)
}

type isEmbeddableMarker struct{}

// Ref fetches a a pointer to the component data. This can be used as a performance optimization to
// not fetch the full component data. It should be used with care, as it allows you to modify the component
// value without dirty checking. You MUST promise not to modify the Value.
type Ref[C spoke.IsComponent[C]] struct {
	Value *C
}

func (Ref[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	idx := result.Builder.FetchComponent(spoke.ComponentTypeOf[C](), false)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke.Filter{}
}

type Option[C spoke.IsComponent[C]] struct {
	value *C
}

func (Option[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	idx := result.Builder.FetchComponent(spoke.ComponentTypeOf[C](), true)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke.Filter{}
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

type OptionMut[C spoke.IsComponent[C]] struct {
	value *C
}

func (OptionMut[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	componentType := spoke.ComponentTypeOf[C]()
	result.Mutable = append(result.Mutable, componentType)

	idx := result.Builder.FetchComponent(spoke.ComponentTypeOf[C](), true)

	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke.Filter{}
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

type Has[C spoke.IsComponent[C]] struct {
	ptr uintptr
}

func (h Has[C]) Exists() bool {
	return h.ptr != 0
}

func (Has[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	componentType := spoke.ComponentTypeOf[C]()

	idx := result.Builder.FetchComponent(componentType, true)
	result.Setters = append(result.Setters, Setter{
		UnsafeFieldOffset:       fieldOffset,
		UnsafeCopyComponentAddr: true,
		ComponentIdx:            idx,
	})

	return spoke.Filter{}
}

type With[C spoke.IsComponent[C]] struct{}

func (With[C]) embeddable(isEmbeddableMarker) {}

func (With[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	return spoke.Filter{
		With: spoke.ComponentTypeOf[C](),
	}
}

type Without[C spoke.IsComponent[C]] struct{}

func (Without[C]) embeddable(isEmbeddableMarker) {}

func (Without[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	return spoke.Filter{
		Without: spoke.ComponentTypeOf[C](),
	}
}

type Changed[C spoke.IsSupportsChangeDetectionComponent[C]] struct{}

func (Changed[C]) embeddable(isEmbeddableMarker) {}

func (Changed[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	return spoke.Filter{
		Changed: spoke.ComponentTypeOf[C](),
	}
}

type Added[C spoke.IsComponent[C]] struct{}

func (Added[C]) embeddable(isEmbeddableMarker) {}

func (Added[C]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	return spoke.Filter{
		Added: spoke.ComponentTypeOf[C](),
	}
}

type Or[A, B Filter] struct{}

func (Or[A, B]) embeddable(isEmbeddableMarker) {}

func (Or[A, B]) applyTo(result *ParsedQuery, fieldOffset uintptr) spoke.Filter {
	var aZero A
	filterA := aZero.applyTo(result, fieldOffset)

	var bZero B
	filterB := bZero.applyTo(result, fieldOffset)

	return spoke.Filter{
		Or: []spoke.Filter{
			filterA,
			filterB,
		},
	}
}

type OrStruct[S any] struct{}

func (OrStruct[S]) embeddable(isEmbeddableMarker) {}

func (OrStruct[S]) applyTo(result *ParsedQuery, baseOffset uintptr) spoke.Filter {
	orStructType := reflect.TypeFor[S]()

	var res spoke.Filter

	for field := range refl.IterFields(orStructType) {
		if field.Name != "_" {
			panic(fmt.Errorf("OrStruct %s field %q should be named \"_\"", orStructType, field.Name))
		}

		fieldOffset := baseOffset + field.Offset

		if !isEmbeddableFilter(field.Type) {
			panic(fmt.Errorf("OrStruct must contain only embeddable filters %s: %s", orStructType, field.Type))
		}

		filter := reflect.New(field.Type).Interface().(Filter)

		f := filter.applyTo(result, fieldOffset)
		if !f.IsZero() {
			res.Or = append(res.Or, f)
		}
	}

	return res
}
