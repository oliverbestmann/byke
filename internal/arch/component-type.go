package arch

import (
	"fmt"
	"hash/maphash"
	"log/slog"
	"maps"
	"reflect"
	"sync/atomic"
	"unsafe"
)

var seed = maphash.MakeSeed()

type UnsafeSetValue func(target unsafe.Pointer, ref ErasedComponent)

type MakeColumn func() Column

type ComponentType struct {
	Name             string
	Type             reflect.Type
	MakeColumn       MakeColumn
	UnsafeSetValue   UnsafeSetValue
	UnsafeSetPointer UnsafeSetValue
	Id               uint32
	Comparable       bool
}

func ComponentTypeOf[C IsComponent[C]]() *ComponentType {
	var zeroValue C

	//goland:noinspection GoDfaNilDereference
	return zeroValue.ComponentType()
}

func (c *ComponentType) New() ErasedComponent {
	return reflect.New(c.Type).Interface().(ErasedComponent)
}

func (c *ComponentType) CopyOf(value ErasedComponent) ErasedComponent {
	target := reflect.New(c.Type)
	target.Elem().Set(reflect.ValueOf(value).Elem())
	return target.Interface().(ErasedComponent)
}

func (c *ComponentType) String() string {
	return c.Name
}

var componentTypes atomic.Pointer[map[unsafe.Pointer]*ComponentType]

func init() {
	// initialize the lookup table
	componentTypes.Store(&map[unsafe.Pointer]*ComponentType{})
}

func ensureComponentType(ptrToType unsafe.Pointer, makeType func(id uint32) *ComponentType) *ComponentType {
	for {
		previousTypes := componentTypes.Load()
		if cached, ok := (*previousTypes)[ptrToType]; ok {
			return cached
		}

		newTypeId := uint32(len(*previousTypes) + 1)

		newType := makeType(newTypeId)

		newTypes := maps.Clone(*previousTypes)
		newTypes[ptrToType] = newType

		if componentTypes.CompareAndSwap(previousTypes, &newTypes) {
			slog.Info(
				"New component type registered",
				slog.String("name", newType.Name),
				slog.Int("id", int(newType.Id)),
			)

			return newType
		}
	}
}

func abiTypePointerTo(t reflect.Type) unsafe.Pointer {
	type eface struct {
		typ, val unsafe.Pointer
	}

	// a reflect.Type is backed by an *rType. The rType contains a abi.Type as
	// its first value. This means, that a *rType can be re-interpreted as *abi.Type
	return (*eface)(unsafe.Pointer(&t)).val
}

func nonComparableComponentTypeOf[C IsComponent[C]]() *ComponentType {
	reflectType := reflect.TypeFor[C]()
	ptrToType := abiTypePointerTo(reflectType)

	if cached, ok := (*componentTypes.Load())[ptrToType]; ok {
		return cached
	}

	if typeHasPaddingBytes(reflectType) {
		fmt.Printf("[warn] type %s contains padding bytes\n", reflectType)
	}

	return ensureComponentType(ptrToType, makeComponentType[C])
}

func comparableComponentTypeOf[C IsComparableComponent[C]]() *ComponentType {
	reflectType := reflect.TypeFor[C]()
	ptrToType := abiTypePointerTo(reflectType)

	if cached, ok := (*componentTypes.Load())[ptrToType]; ok {
		return cached
	}

	if typeHasPaddingBytes(reflectType) {
		fmt.Printf("[warn] type %s contains padding bytes\n", reflectType)
	}

	return ensureComponentType(ptrToType, func(id uint32) *ComponentType {
		ty := makeComponentType[C](id)
		ty.MakeColumn = MakeComparableColumnOf[C](ty)
		ty.Comparable = true
		return ty
	})
}

func makeComponentType[C IsComponent[C]](id uint32) *ComponentType {
	reflectType := reflect.TypeFor[C]()

	ty := &ComponentType{
		Id:   id,
		Type: reflectType,
		Name: reflectType.String(),
	}

	ty.MakeColumn = MakeColumnOf[C](ty)

	ty.UnsafeSetValue = unsafeCopyComponentValue[C]
	ty.UnsafeSetPointer = unsafeSetComponentPointer[C]

	return ty
}

func unsafeCopyComponentValue[C ErasedComponent](target unsafe.Pointer, value ErasedComponent) {
	// target points to a C
	*(*C)(target) = *any(value).(*C)
}

func unsafeSetComponentPointer[C ErasedComponent](target unsafe.Pointer, value ErasedComponent) {
	// target points to a variable of type *C
	*(**C)(target) = any(value).(*C)
}
