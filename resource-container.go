package byke

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
)

type resourceValue[T any] struct {
	// Value holds a pointer to the actual resource value
	Value   *T
	IsValid bool
}

func (r *resourceValue[T]) Get() (any, bool) {
	return r.Value, r.IsValid
}

func (r *resourceValue[T]) Set(value T) {
	*r.Value = value
	r.IsValid = true
}

func (r *resourceValue[T]) Invalidate() {
	r.IsValid = false
}

type ErasedResource interface {
	Get() (any, bool)
}

type resourceContainer map[reflect.Type]interface {
	ErasedResource
	Invalidate()
}

// InsertResource inserts a new resource into the world.
// The resource should be provided as a non-pointer type.
//
// If the resource does not yet exist, a new value of the resources type will
// be allocated on the heap and the value provided will be copied into that memory location.
//
// If the world already contains a resource of the same type, this value will
// just be updated with the newly provided one.
func (rc *resourceContainer) InsertResource[T any](resource T) {
	resType := reflect.TypeFor[T]()

	if existing, ok := (*rc)[resType]; ok {
		// update existing value in place
		existing.(*resourceValue[T]).Set(resource)
		return
	}

	if resType == reflect.TypeFor[any]() {
		panic(errors.New("cannot insert resource of type 'any'"))
	}

	slog.Debug("Inserting new resource", slog.String("type", resType.String()))

	(*rc)[resType] = &resourceValue[T]{
		Value:   new(resource),
		IsValid: true,
	}
}

// RemoveResource removes a resource previously added with InsertResource.
func (rc *resourceContainer) RemoveResource(resourceType reflect.Type) {
	if existing, ok := (*rc)[resourceType]; ok {
		existing.Invalidate()
	}

	// delete((*rc), resourceType)
}

// Resource returns a pointer to the resource of the given reflect type.
// The type must be the non-pointer type of the resource, i.e. the type of the resource
// as it was passed to InsertResource.
func (rc *resourceContainer) Resource(ty reflect.Type) (AnyPtr, bool) {
	resValue, ok := (*rc)[ty]
	if !ok {
		return nil, false
	}

	return resValue.Get()
}

func (rc *resourceContainer) ResourceOf[T any]() (*T, bool) {
	value, ok := rc.Resource(reflect.TypeFor[T]())
	if !ok {
		return nil, false
	}

	return value.(*T), true
}

func (rc *resourceContainer) RequireResourceOf[T any]() *T {
	res, ok := rc.ResourceOf[T]()
	if !ok {
		var tZero T
		panic(fmt.Errorf("resource of type %T not found", tZero))
	}

	return res
}

func (rc *resourceContainer) RefToResource(ty reflect.Type) (ErasedResource, bool) {
	resValue, ok := (*rc)[ty]
	return resValue, ok
}
