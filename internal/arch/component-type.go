package arch

import (
	"fmt"
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
	Name       string
	Type       reflect.Type
	MakeColumn MakeColumn
	SetValue   SetValue
	Comparable bool
}

func (c *ComponentType) String() string {
	return c.Type.String()
}

func (c *ComponentType) PtrType() reflect.Type {
	return reflect.PointerTo(c.Type)
}

func (c *ComponentType) New() ErasedComponent {
	return reflect.New(c.Type).Interface().(ErasedComponent)
}

var componentTypes = map[unsafe.Pointer]*ComponentType{}

func abiTypePointerTo(t reflect.Type) unsafe.Pointer {
	type eface struct {
		typ, val unsafe.Pointer
	}

	return (*eface)(unsafe.Pointer(&t)).val
}

func ComponentTypeOf[C IsComponent[C]]() *ComponentType {
	var zeroComponent C

	//goland:noinspection GoDfaNilDereference
	return zeroComponent.ComponentType()
}

func nonComparableComponentTypeOf[C IsComponent[C]]() *ComponentType {
	reflectType := reflect.TypeFor[C]()

	ptrToType := abiTypePointerTo(reflectType)
	ty, ok := componentTypes[ptrToType]

	if !ok {
		if typeHasPaddingBytes(reflectType) {
			fmt.Printf("[warn] type %s contains padding bytes\n", reflectType)
		}

		ty = &ComponentType{
			Id:   int64(len(componentTypes) + 1),
			Type: reflectType,
			Name: reflectType.String(),
		}

		ty.MakeColumn = MakeColumnOf[C](ty)

		ty.SetValue = func(target any, source ErasedComponent) {
			value := any(source).(*C)

			// target value must be either a pointer or a pointer to a pointer
			switch ptrToTarget := target.(type) {
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

func comparableComponentTypeOf[C IsComparableComponent[C]]() *ComponentType {
	ptrToType := abiTypePointerTo(reflect.TypeFor[C]())
	ty, ok := componentTypes[ptrToType]

	if !ok {
		ty = nonComparableComponentTypeOf[C]()
		ty.MakeColumn = MakeComparableColumnOf[C](ty)
		ty.Comparable = true
	}

	return ty
}
