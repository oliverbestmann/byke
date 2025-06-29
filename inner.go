package ecs

import "reflect"

type InnerType[S any] struct{}

func (InnerType[S]) innerType() reflect.Type {
	return reflect.TypeFor[S]()
}
