package arch

import (
	"hash/maphash"
	"reflect"
)

var seed = maphash.MakeSeed()

type HashOf func(value ErasedComponent) HashValue

type CopyTo func(from, to ErasedComponent)

type MakeColumn func() Column

type ComponentType struct {
	Id         int64
	Type       reflect.Type
	MakeColumn MakeColumn
	hashOf     HashOf
	copyTo     CopyTo
}

func (c *ComponentType) String() string {
	return c.Type.String()
}

func (c *ComponentType) IsComparable() bool {
	return c.hashOf != nil
}

func (c *ComponentType) PtrType() reflect.Type {
	return reflect.PointerTo(c.Type)
}

func (c *ComponentType) New() ErasedComponent {
	return reflect.New(c.Type).Interface().(ErasedComponent)
}

func (c *ComponentType) CopyValue(from, to ErasedComponent) {
	c.copyTo(from, to)
}

func (c *ComponentType) MaybeHashOf(component ErasedComponent) HashValue {
	if c.hashOf != nil {
		c.hashOf(component)
	}

	return 0
}

var componentTypes = map[reflect.Type]*ComponentType{}

func ComponentTypeOf[C IsComponent[C]]() *ComponentType {
	ty, ok := componentTypes[reflect.TypeFor[C]()]

	if !ok {
		ty = &ComponentType{
			Id:   int64(len(componentTypes) + 1),
			Type: reflect.TypeFor[C](),

			MakeColumn: columnConstructorOf[C](ty),

			copyTo: func(from, to ErasedComponent) {
				ptrToFromValue := any(from).(*C)
				ptrToToValue := any(to).(*C)
				*ptrToFromValue = *ptrToToValue
			},
		}

		componentTypes[ty.Type] = ty
	}

	return ty
}

func ComparableComponentTypeOf[C IsComparableComponent[C]]() *ComponentType {
	ty, ok := componentTypes[reflect.TypeFor[C]()]

	if !ok {
		ty = ComponentTypeOf[C]()

		ty.hashOf = func(value ErasedComponent) HashValue {
			ptrToValue := any(value).(*C)
			return HashValue(maphash.Comparable[C](seed, *ptrToValue))
		}
	}

	return ty
}
