package arch

import (
	"hash/maphash"
	"reflect"
	"unsafe"
)

var seed = maphash.MakeSeed()

type HashOf func(value ErasedComponent) HashValue

type SetValue func(target any, source ErasedComponent)

type MakeColumn func() Column

type ComponentType struct {
	Id         int64
	Type       reflect.Type
	MakeColumn MakeColumn
	SetValue   SetValue
	hashOf     HashOf
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

func (c *ComponentType) MaybeHashOf(component ErasedComponent) HashValue {
	if c.hashOf != nil {
		c.hashOf(component)
	}

	return 0
}

var componentTypes = map[uintptr]*ComponentType{}

func abiTypePointerTo(t reflect.Type) uintptr {
	type eface struct {
		typ, val unsafe.Pointer
	}

	return uintptr((*eface)(unsafe.Pointer(&t)).val)
}

func ComponentTypeOf[C IsComponent[C]]() *ComponentType {
	ptrToType := abiTypePointerTo(reflect.TypeFor[C]())
	ty, ok := componentTypes[ptrToType]

	if !ok {
		ty = &ComponentType{
			Id:   int64(len(componentTypes) + 1),
			Type: reflect.TypeFor[C](),
		}

		ty.MakeColumn = MakeColumnOf[C](ty)

		ty.SetValue = func(target any, source ErasedComponent) {
			value := any(source).(*C)

			// target value must be either a pointer or a pointer to a pointer
			switch ptrToTarget := any(target).(type) {
			case *C:
				*ptrToTarget = *value
			case **C:
				*ptrToTarget = value
			}
		}

		componentTypes[ptrToType] = ty
	}

	return ty
}

func ComparableComponentTypeOf[C IsComparableComponent[C]]() *ComponentType {
	ptrToType := abiTypePointerTo(reflect.TypeFor[C]())
	ty, ok := componentTypes[ptrToType]

	if !ok {
		ty = ComponentTypeOf[C]()

		ty.hashOf = func(value ErasedComponent) HashValue {
			ptrToValue := any(value).(*C)
			return HashValue(maphash.Comparable[C](seed, *ptrToValue))
		}

		ty.MakeColumn = MakeComparableColumnOf[C](ty)
	}

	return ty
}
