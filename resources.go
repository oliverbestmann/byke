package byke

import (
	"fmt"
	"reflect"
)

type resourceSystemParamState struct {
	typ   reflect.Type
	world *World

	// true if the system wants the pointer type
	mutable bool
}

func makeResourceSystemParamState(world *World, typ reflect.Type) SystemParamState {
	r := resourceSystemParamState{
		world:   world,
		mutable: typ.Kind() == reflect.Pointer,
		typ:     typ,
	}

	if r.mutable {
		// if typ is a pointer, we reduce it to the type itself.
		r.typ = r.typ.Elem()
	}

	return r
}

func (r resourceSystemParamState) getValue(systemContext) reflect.Value {
	ptrToValue, ok := r.world.Resource(r.typ)
	if !ok {
		panic(fmt.Sprintf("Resource of type %s does not exist in world", r.typ))
	}

	if r.mutable {
		return reflect.ValueOf(ptrToValue)
	}

	return reflect.ValueOf(ptrToValue).Elem()
}

func (r resourceSystemParamState) cleanupValue() {
}

func (r resourceSystemParamState) valueType() reflect.Type {
	if r.mutable {
		return reflect.PointerTo(r.typ)
	} else {
		return r.typ
	}
}

// Res provides a SystemParam to inject a resource at runtime.
//
// This is currently the same as just declaring the resource type directly as a parameter.
// In the future, the Res type may offer additional information, such as resource change tracking.
type Res[T any] struct {
	Value T
	world *World
}

func (r *Res[T]) init(world *World) SystemParamState {
	r.world = world
	return r
}

func (r *Res[T]) getValue(systemContext) reflect.Value {
	lookupType := reflect.TypeFor[T]()
	if lookupType.Kind() == reflect.Pointer {
		lookupType = lookupType.Elem()
	}

	resValue, ok := r.world.Resource(lookupType)
	if !ok {
		panic(fmt.Sprintf("no value for resource of type %s", lookupType))
	}

	r.setValue(resValue)

	return reflect.ValueOf(r).Elem()
}

func (r *Res[T]) cleanupValue() {
}

func (r *Res[T]) valueType() reflect.Type {
	return reflect.TypeFor[Res[T]]()
}

func (r *Res[T]) setValue(value any) {
	// the value we get is always a pointer to the resource
	switch value := value.(type) {
	case T:
		// this is the case if T is a pointer to a resource
		r.Value = value

	case *T:
		// this is the case if T is the actual resource, we do
		// a copy in this case
		r.Value = *value
	}
}

// ResOption allows to inject a resource as a system param if it exists in the world.
// If the resource does not exist, the system will still run but a zero ResOption is injected.
type ResOption[T any] struct {
	Value *T
	world *World
}

func (r *ResOption[T]) init(world *World) SystemParamState {
	r.world = world
	return r
}

func (r *ResOption[T]) getValue(systemContext) reflect.Value {
	lookupType := reflect.TypeFor[T]()

	resValue, ok := r.world.Resource(lookupType)
	if !ok {
		r.Value = nil
	} else {
		r.Value = resValue.(*T)
	}

	return reflect.ValueOf(r).Elem()
}

func (r *ResOption[T]) cleanupValue() {
}

func (r *ResOption[T]) valueType() reflect.Type {
	return reflect.TypeFor[ResOption[T]]()
}
