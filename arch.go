package byke

import (
	"github.com/oliverbestmann/byke/internal/query"
	spoke2 "github.com/oliverbestmann/byke/spoke"
)

// EntityId uniquely identifies an entity in a World.
type EntityId = spoke2.EntityId

// IsComponent can be used in a type parameter to ensure that type T is a Component type.
//
// To implement the IsComponent interface for a type, you must embed the Component type.
type IsComponent[T any] = spoke2.IsComponent[T]

// IsImmutableComponent indicates that a component is immutable and can not be queried
// using pointer access. Immutable components can be updated by inserting a new copy of the
// same component into an entity using Command.
//
// To implement the IsImmutableComponent interface for a type, you must embed the ImmutableComponent type.
type IsImmutableComponent[T IsComponent[T]] = spoke2.IsImmutableComponent[T]

// IsComparableComponent indicates that a component is comparable. Only comparable components
// and immutable components can be used with the Changed query filter.
// At the time of writing, comparable components have a performance overhead when queried
// by pointer.
//
// To implement the IsComparableComponent interface for a type, you must embed the ComparableComponent type.
type IsComparableComponent[T IsComponent[T]] = spoke2.IsComparableComponent[T]

// Component is a zero sized type that may be embedded into a struct to turn that
// struct into a component (see IsComponent).
type Component[T IsComponent[T]] = spoke2.Component[T]

// ImmutableComponent is a zero sized type that may be embedded into a struct to turn that
// struct into an immutable component (see IsComponent).
type ImmutableComponent[T spoke2.IsImmutableComponent[T]] = spoke2.ImmutableComponent[T]

// ComparableComponent is a zero sized type that may be embedded into a struct to turn that
// struct into a comparable component (see IsComponent).
type ComparableComponent[T IsComparableComponent[T]] = spoke2.ComparableComponent[T]

// ErasedComponent indicates a type erased Component value.
//
// Values given to the consumer of byke of this type are usually pointers,
// even though the interface is actually implemented directly on the component type.
type ErasedComponent = spoke2.ErasedComponent

// Option is a query parameter that fetches a given Component of type T
// if it exists on an entity.
type Option[C IsComponent[C]] = query.Option[C]

// OptionMut is a query parameter that fetches for a pointer to a Component of type T
// if it exists on an entity.
type OptionMut[C IsComponent[C]] = query.OptionMut[C]

// Has is a query parameter that does not fetch the actual Component value of type T,
// but rather just indicates if a component of such type exists on the type. Currently
// it does not provide a performance boost over using an Option.
type Has[C IsComponent[C]] = query.Has[C]

// With is a query filter that constraints the entities queried to include only
// entities that have a Component of type T.
type With[C IsComponent[C]] = query.With[C]

// Without is a query filter that constraints the entities queried to include only
// entities that do not have a Component of type T.
type Without[C IsComponent[C]] = query.Without[C]

// Added is a query filter that constraints the entities matched to include only
// entities that have added a component of type C since the last tick that the
// system owning the Query ran.
type Added[C IsComponent[C]] = query.Added[C]

// Changed is a query filter that constraints the entities matched to include only
// entities that have a changed Component value of type C since the last tick that the
// system owning the Query ran.
//
// Change detection currently works by hashing the component value. As such, matching
// for changed component values is not 100% foolproof in case of hash collisions.
type Changed[C spoke2.IsSupportsChangeDetectionComponent[C]] = query.Changed[C]

// Or is a query filter that allows you to combine two query filters with a local 'or'.
// Simply adding multiple filters to a query requires all filters to match. Using Or
// you can build a query, where just one of multiple filter need to match
type Or[A, B query.Filter] = query.Or[A, B]
